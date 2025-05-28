package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type WhatsAppChatMessage struct {
	Role            string
	Text            string
	ToolCallID      string
	ToolCallMessage openai.ChatCompletionMessage
	FileBase64      string
	FileMimetype    string
}

type WhatsAppChat struct {
	Number               string
	Messages             []WhatsAppChatMessage
	LastInteractionTime  time.Time
	OpenAIStack          openai.ChatCompletion
	ToolCall             openai.ChatCompletionMessageToolCall
	ToolCallMessageParam openai.ChatCompletionMessageParamUnion
	ToolCallMessage      openai.ChatCompletionMessage
	AllowSendReceipt     bool
	Fullname             string
	Order                OrdemDeCompra
	Receipt              EvolutionMedia
}

func (chat *WhatsAppChat) SendToOpenAI(message string, messageRole string, file *EvolutionMedia) {
	client := openai.NewClient(
		option.WithAPIKey(Vault.OpenAIApiKey),
	)
	endConversation := false

	chatMessage := WhatsAppChatMessage{
		Role: messageRole,
		Text: message,
	}

	if file != nil {
		chatMessage.FileBase64 = file.Base64
		chatMessage.FileMimetype = file.MimeType
		chat.Receipt = *file
	}

	chat.Messages = append(chat.Messages, chatMessage)
	chat.LastInteractionTime = time.Now()

	// TODO Wait for X seconds before send (buffer messages)

	finishCheckout := openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "finalizar_checkout",
			Strict:      openai.Bool(true),
			Description: openai.String("Deve ser chamado após finalizar a escolha dos produtos e uma forma de pagamento. Ou seja, assim que você retonar a mensagem \"Pedido confirmado\". Não precisa validar o comprovante pix para chamar esta função"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"required": []string{
					"produtos",
					"valor_total",
					"nome_completo",
					"endereco",
					"forma_de_pagamento",
				},
				"properties": map[string]interface{}{
					"produtos": map[string]interface{}{
						"type":        "array",
						"description": "Lista de produtos no carrinho",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"id_produto": map[string]string{
									"type":        "string",
									"description": "Identificador único do produto",
								},
								"nome_produto": map[string]string{
									"type":        "string",
									"description": "Nome do produto",
								},
								"quantidade": map[string]string{
									"type":        "number",
									"description": "Quantidade do produto",
								},
								"valor": map[string]string{
									"type":        "string",
									"description": "Preço unitário do produto",
								},
								"detalhes": map[string]string{
									"type":        "string",
									"description": "Sabor e qualquer outro detalhe acrescentado durante a conversa",
								},
							},
							"additionalProperties": false,
							"required": []string{
								"id_produto",
								"nome_produto",
								"quantidade",
								"valor",
								"detalhes",
							},
						},
					},
					"valor_total": map[string]string{
						"type":        "string",
						"description": "Valor total da compra, somando todos os produtos",
					},
					"nome_completo": map[string]string{
						"type":        "string",
						"description": "Nome completo do usuário",
					},
					"endereco": map[string]string{
						"type":        "string",
						"description": "Endereço de entrega dos produtos",
					},
					"forma_de_pagamento": map[string]interface{}{
						"type":        "string",
						"description": "A forma de pagamento escolhida pelo usuário",
						"enum": []string{
							"cartao_de_credito",
							"cartao_de_debito",
							"dinheiro",
							"pix",
						},
					},
				},
				"additionalProperties": false,
			},
		},
	}

	sendCatalog := openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "enviar_catalogo",
			Strict:      openai.Bool(true),
			Description: openai.String("Enviar o catálogo para o usuário quando pedido. Deverá ser chamado quando o usuário pedir o catálogo."),
			Parameters: openai.FunctionParameters{
				"type":                 "object",
				"required":             []string{},
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(Vault.SystemMessage),
	}

	if chat.Fullname != "" && len(chat.Messages) == 0 {
		chat.Messages = append(chat.Messages, WhatsAppChatMessage{
			Role: "developer",
			Text: fmt.Sprintf("Este usuário já interagiu conosco antes e o nome completo dele é %s", chat.Fullname),
		})
	}

	for _, msg := range chat.Messages {
		switch role := msg.Role; role {
		case "user":
			var message openai.ChatCompletionMessageParamUnion

			if msg.FileBase64 != "" {
				if msg.FileMimetype == "image/jpeg" || msg.FileMimetype == "image/jpg" {
					message = openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
						openai.TextContentPart(msg.Text),
						openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
							URL: fmt.Sprintf("data:%s;base64,%s", msg.FileMimetype, msg.FileBase64),
						}),
					})
				}

				if msg.FileMimetype == "application/pdf" {
					message = openai.DeveloperMessage("Arquivo recebido é pdf, não é necessário realizar validação.")
					// message = openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
					// 	openai.TextContentPart(msg.Text),
					// 	openai.FileContentPart(openai.ChatCompletionContentPartFileFileParam{
					// 		Filename: openai.String("Comprovante.pdf"),
					// 		FileData: openai.String(fmt.Sprintf("data:%s;base64,%s", msg.FileMimetype, msg.FileBase64)),
					// 	}),
					// })
				}
			} else {
				message = openai.UserMessage(msg.Text)
			}

			messages = append(messages, message)
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Text))
		case "developer":
			messages = append(messages, openai.DeveloperMessage(msg.Text))
		case "tool":
			messages = append(messages, openai.ToolMessage(msg.Text, msg.ToolCallID))
		case "function_call":
			messages = append(messages, msg.ToolCallMessage.ToParam())
		}
	}

	params := openai.ChatCompletionNewParams{
		Messages: messages,
		Tools: []openai.ChatCompletionToolParam{
			finishCheckout,
			sendCatalog,
		},
		Temperature: openai.Float(1.0),
		Model:       "gpt-4.1",
		Seed:        openai.Int(0),
	}

	res, err := client.Chat.Completions.New(context.Background(), params)
	failOnError(err, "Can't send messages to OpenAI")
	chat.OpenAIStack = *res

	// check enviar catalogo tool call
	if chat.HasToolCallInLastMessage("enviar_catalogo", true) {
		chat.SendDocToWhatsapp(Vault.CatalogAttachment, "application/pdf", "Catálogo.pdf")
		res, err = chat.SendToolCallResponse("catálogo enviado", &params, &client)
		failOnError(err, "Can't send messages to OpenAI")
		chat.AllowSendReceipt = true
	}

	// check finalizar_checkout tool call
	if chat.HasToolCallInLastMessage("finalizar_checkout", true) {
		err = GetToolArgs(chat.ToolCall, &chat.Order)
		failOnError(err, "Can't parse tool args")

		res, err = chat.SendToolCallResponse("pedido recebido, aguardando comprovante", &params, &client)
		failOnError(err, "Can't send messages to OpenAI")

		chat.Fullname = chat.Order.NomeCompleto
		var orderStr string
		var paymentMethod string

		for _, product := range chat.Order.Produtos {
			productDetail := ""
			if product.Detalhes != "" {
				productDetail = fmt.Sprintf(" (%s)", product.Detalhes)
			}
			orderStr = fmt.Sprintf("%s\n%d %s%s, %s", orderStr, product.Quantidade, product.NomeProduto, productDetail, product.Valor)
		}

		if chat.Order.FormaDePagamento == "pix" {
			mediaType := "image"
			if chat.Receipt.MediaType == "documentMessage" {
				mediaType = "document"
			}
			SendMediaToNumber(
				Vault.OwnerNumber,
				[]byte(chat.Receipt.Base64),
				mediaType,
				chat.Receipt.MimeType,
				fmt.Sprintf("Comprovante de %s", chat.Fullname),
				false,
			)
		}

		switch chat.Order.FormaDePagamento {
		case "pix":
			paymentMethod = "Pix"
		case "cartao_de_credito":
			paymentMethod = "Cartão de crédito"
		case "cartao_de_debito":
			paymentMethod = "Cartão de débito"
		case "dinheiro":
			paymentMethod = "Dinheiro"
		}

		SendMessageToNumber(
			Vault.OwnerNumber,
			fmt.Sprintf(
				"Pedido de %s no valor total de %s\n\n%s\n\nEndereço de entrega: %s\nForma de pagamento: %s\nhttps://wa.me/%s",
				chat.Fullname,
				chat.Order.ValorTotal,
				orderStr,
				chat.Order.Endereco,
				paymentMethod,
				chat.Number,
			),
		)

		endConversation = true
	}

	chat.OpenAIStack = *res
	lastMessage := chat.OpenAIStack.Choices[len(chat.OpenAIStack.Choices)-1]
	if lastMessage.Message.Role == "assistant" {
		chat.SendMessageToWhatsApp(lastMessage.Message.Content)
	}

	chat.Messages = append(chat.Messages, WhatsAppChatMessage{
		Role: string(lastMessage.Message.Role),
		Text: string(lastMessage.Message.Content),
	})

	if endConversation {
		chat.SaveToLog()
		chat.Clear()
	}

	// suspend last interaction chat on db
	chat.Suspend()
}

func (chat *WhatsAppChat) SendToolCallResponse(message string, params *openai.ChatCompletionNewParams, client *openai.Client) (*openai.ChatCompletion, error) {
	params.Messages = append(params.Messages, chat.ToolCallMessageParam)
	chat.Messages = append(chat.Messages, WhatsAppChatMessage{
		Role:            "function_call",
		ToolCallMessage: chat.ToolCallMessage,
	})

	params.Messages = append(params.Messages, openai.ToolMessage(message, chat.ToolCall.ID))
	chat.Messages = append(chat.Messages, WhatsAppChatMessage{
		Role:       "tool",
		Text:       message,
		ToolCallID: chat.ToolCall.ID,
	})

	return client.Chat.Completions.New(context.Background(), *params)
}

func (chat WhatsAppChat) SendMessageToWhatsApp(message string) {
	body, _ := json.Marshal(map[string]string{
		"number": chat.Number,
		"text":   message,
	})

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/message/sendText/%s", Vault.EvolutionURL, Vault.InstanceID),
		bytes.NewBuffer(body),
	)
	failOnError(err, "Can't create request to evolution api")

	req.Header.Add("apikey", Vault.EvolutionToken)
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	failOnError(err, "Can't send message to evolution api")

	_, err = io.ReadAll(res.Body)
	failOnError(err, "Can't parse response")
	fmt.Printf("Message sent to whatsapp (%s): %s\n", chat.Number, message)
}

func (chat WhatsAppChat) SendDocToWhatsapp(file []byte, mimeType string, fileName string) {
	doc := base64.StdEncoding.EncodeToString(file)
	body, _ := json.Marshal(map[string]string{
		"number":    chat.Number,
		"mimetype":  mimeType,
		"mediatype": "document",
		"fileName":  fileName,
		"media":     doc,
	})

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/message/sendMedia/%s", Vault.EvolutionURL, Vault.InstanceID),
		bytes.NewBuffer(body),
	)
	failOnError(err, "Can't create request to evolution api")

	req.Header.Add("apikey", Vault.EvolutionToken)
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	failOnError(err, "Can't parse response")

	resBody, _ := io.ReadAll(res.Body)
	fmt.Println(string(resBody))
	fmt.Println("Doc sent to whatsapp")
}

func (chat WhatsAppChat) Suspend() {
	marshed, err := json.Marshal(chat)
	failOnError(err, "Failed to marshal chat")

	var found int
	err = Vault.PGX.QueryRow(context.Background(), "SELECT COUNT(phone_number) found FROM suspended_chats WHERE phone_number = $1", chat.Number).Scan(&found)
	failOnError(err, "Can't scan for suspended_chats")

	if found < 1 {
		_, err = Vault.PGX.Exec(context.Background(), "INSERT INTO suspended_chats (phone_number, data) VALUES ($1, $2)", chat.Number, marshed)
		failOnError(err, "Can't insert new record to suspended_chats")
	} else {
		_, err = Vault.PGX.Exec(context.Background(), "UPDATE suspended_chats SET data = $2 WHERE phone_number = $1", chat.Number, marshed)
		failOnError(err, "Can't update the record in suspended_chats")
	}
}

func (chat WhatsAppChat) SaveToLog() {
	marshed, err := json.Marshal(chat)
	failOnError(err, "Failed to marshal chat")

	_, err = Vault.PGX.Exec(context.Background(), "INSERT INTO chat_logs (phone_number, data) VALUES ($1, $2)", chat.Number, marshed)
	failOnError(err, "Can't save log")
}

func (chat *WhatsAppChat) Clear() {
	chat.Messages = []WhatsAppChatMessage{}
	chat.AllowSendReceipt = false
}

func (chat *WhatsAppChat) HasToolCallInLastMessage(functionName string, write bool) bool {
	if len(chat.Messages) > 0 {
		lastMessage := chat.OpenAIStack.Choices[len(chat.OpenAIStack.Choices)-1]
		if len(lastMessage.Message.ToolCalls) > 0 {
			for _, toolCall := range lastMessage.Message.ToolCalls {
				if toolCall.Function.Name == functionName {

					if write {
						chat.ToolCall = toolCall
						chat.ToolCallMessage = lastMessage.Message
						chat.ToolCallMessageParam = lastMessage.Message.ToParam()
					}
					return true
				}
			}
		}
	}

	return false
}

func SendMediaToNumber(number string, file []byte, mediatype string, mimeType string, fileName string, encodeBase64 bool) {
	doc := string(file)
	if encodeBase64 {
		doc = base64.StdEncoding.EncodeToString(file)
	}

	body, _ := json.Marshal(map[string]string{
		"number":    number,
		"mimetype":  mimeType,
		"mediatype": mediatype,
		"fileName":  fileName,
		"media":     doc,
	})

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/message/sendMedia/%s", Vault.EvolutionURL, Vault.InstanceID),
		bytes.NewBuffer(body),
	)
	failOnError(err, "Can't create request to evolution api")

	req.Header.Add("apikey", Vault.EvolutionToken)
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	failOnError(err, "Can't parse response")

	resBody, _ := io.ReadAll(res.Body)
	fmt.Println(string(resBody))
	fmt.Println("Doc sent to whatsapp")
}

func SendMessageToNumber(number string, message string) {
	body, _ := json.Marshal(map[string]string{
		"number": number,
		"text":   message,
	})

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/message/sendText/%s", Vault.EvolutionURL, Vault.InstanceID),
		bytes.NewBuffer(body),
	)
	failOnError(err, "Can't create request to evolution api")

	req.Header.Add("apikey", Vault.EvolutionToken)
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	failOnError(err, "Can't send message to evolution api")

	_, err = io.ReadAll(res.Body)
	failOnError(err, "Can't parse response")
	fmt.Printf("Message sent to whatsapp (%s): %s\n", number, message)
}

func GetMediaBase64(key string) EvolutionMedia {
	body, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"key": map[string]string{
				"id": key,
			},
		},
		"convertToMp4": false,
	})

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/chat/getBase64FromMediaMessage/%s", Vault.EvolutionURL, Vault.InstanceID),
		bytes.NewBuffer(body),
	)
	failOnError(err, "Can't create request to evolution api")

	req.Header.Add("apikey", Vault.EvolutionToken)
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	failOnError(err, "Can't send message to evolution api")

	resp, err := io.ReadAll(res.Body)
	failOnError(err, "Can't parse response")

	var respData EvolutionMedia
	json.Unmarshal(resp, &respData)

	return respData
}

func GetToolArgs[T any](toolCall openai.ChatCompletionMessageToolCall, to *T) error {
	var args T
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
	if err != nil {
		return err
	}

	*to = args
	return nil
}
