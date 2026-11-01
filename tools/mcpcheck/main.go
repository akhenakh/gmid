// Command mcpcheck drives a running gmid MCP server for smoke tests.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	var (
		url  = flag.String("url", "http://127.0.0.1:8765/mcp", "streamable HTTP endpoint")
		tool = flag.String("tool", "gmid_status", "tool to call")
		args = flag.String("args", "{}", "JSON arguments")
		list = flag.Bool("list", false, "list tools and exit")
		out  = flag.String("out", "", "write an image result to this file")
	)
	flag.Parse()

	tr, err := transport.NewStreamableHTTP(*url)
	if err != nil {
		fail(err)
	}
	defer func() { _ = tr.Close() }()
	c := client.NewClient(tr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := c.Start(ctx); err != nil {
		fail(err)
	}
	init, err := c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "gmid-mcpcheck", Version: "0.1"},
		},
	})
	if err != nil {
		fail(err)
	}
	fmt.Printf("connected: %s %s\n", init.ServerInfo.Name, init.ServerInfo.Version)

	if *list {
		tools, err := c.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			fail(err)
		}
		for _, t := range tools.Tools {
			fmt.Println("-", t.Name)
		}
		return
	}

	var argv any
	if err := json.Unmarshal([]byte(*args), &argv); err != nil {
		fail(err)
	}
	res, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: *tool, Arguments: argv},
	})
	if err != nil {
		fail(err)
	}
	for _, content := range res.Content {
		switch v := content.(type) {
		case mcp.TextContent:
			fmt.Println(v.Text)
		case mcp.ImageContent:
			fmt.Printf("[image %s, %d base64 bytes]\n", v.MIMEType, len(v.Data))
			if *out != "" {
				data, err := base64.StdEncoding.DecodeString(v.Data)
				if err != nil {
					fail(err)
				}
				if err := os.WriteFile(*out, data, 0o644); err != nil {
					fail(err)
				}
				fmt.Println("saved", *out)
			}
		default:
			fmt.Printf("%#v\n", content)
		}
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "mcpcheck:", err)
	os.Exit(1)
}
