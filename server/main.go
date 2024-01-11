package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/juliotorresmoreno/turn/common"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Context struct {
	UUID     string
	Response chan *common.Response
}

var data = map[string]interface{}{}

func main() {
	r := gin.New()
	t := gin.New()

	event := make(chan *common.Event)
	response := make(chan *common.Response)
	register := make(chan *Context)

	go func() {
		for {
			select {
			case register := <-register:
				data[register.UUID] = register.Response
			case response := <-response:
				if value, ok := data[response.UUID]; ok {
					value.(chan *common.Response) <- response
				}
			case event := <-event:
				if connection == nil {
					continue
				}
				buff, err := json.Marshal(event)
				if err != nil {
					continue
				}
				err = connection.WriteMessage(websocket.BinaryMessage, buff)
				if err != nil {
					continue
				}
			}
		}
	}()

	r.Use(func(ctx *gin.Context) {
		response := make(chan *common.Response)
		UUID := uuid.New().String()
		register <- &Context{
			UUID:     UUID,
			Response: response,
		}
		event <- &common.Event{
			UUID:   UUID,
			URL:    ctx.Request.URL.String(),
			Method: ctx.Request.Method,
			Header: http.Header{},
			Body:   []byte{},
		}
		r := <-response
		ctx.Status(r.StatusCode)
		for h := range r.Header {
			ctx.Header(h, r.Header.Get(h))
		}

		ctx.Writer.Write(r.Body)
		ctx.Writer.Flush()

	})

	t.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer func() {
			conn.Close()
			connection = nil
		}()
		connection = conn

		for {
			message := &common.Response{}
			_, p, err := conn.ReadMessage()
			if err != nil {
				fmt.Println(err)
				return
			}
			json.Unmarshal(p, message)
			response <- message
		}
	})

	go t.Run(":8081")
	r.Run(":8080")
}

var connection *websocket.Conn
