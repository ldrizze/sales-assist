package main

import (
	"encoding/json"
)

func (chat *WhatsAppChat) CheckForFinishOrder() bool {
	if len(chat.Messages) > 0 {
		lastMessage := chat.OpenAIStack.Choices[len(chat.OpenAIStack.Choices)-1]
		if len(lastMessage.Message.ToolCalls) > 0 {
			if lastMessage.Message.ToolCalls[0].Function.Name == "finalizar_checkout" {
				var args OrdemDeCompra
				err := json.Unmarshal([]byte(lastMessage.Message.ToolCalls[0].Function.Arguments), &args)
				if err != nil {
					failOnError(err, "Can't unwrap tool call json")
				}

				chat.Order = args
				chat.ToolCall = lastMessage.Message.ToolCalls[0]
				chat.ToolCallMessage = lastMessage.Message
				chat.ToolCallMessageParam = lastMessage.Message.ToParam()
				return true
			}
		}
	}

	return false
}
