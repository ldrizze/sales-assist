package main

import (
	"github.com/jackc/pgx/v5"
)

type AppVault struct {
	InstanceID           string
	OpenAIApiKey         string
	RabbitMQExchangeName string
	EvolutionToken       string
	EvolutionURL         string
	Conversations        map[string]*WhatsAppChat
	SystemMessage        string
	EventHarvestList     []*WhatsAppChat
	PGX                  *pgx.Conn
	OwnerNumber          string
	EnableForMe          bool
	CatalogAttachment    []byte
}

var Vault AppVault
