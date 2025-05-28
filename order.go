package main

type Produto struct {
	IdProduto   string `json:"id_produto"`
	NomeProduto string `json:"nome_produto"`
	Quantidade  int    `json:"quantidade"`
	Valor       string `json:"valor"`
	Detalhes    string `json:"detalhes"`
}

type OrdemDeCompra struct {
	Produtos         []Produto `json:"produtos"`
	ValorTotal       string    `json:"valor_total"`
	NomeCompleto     string    `json:"nome_completo"`
	Endereco         string    `json:"endereco"`
	FormaDePagamento string    `json:"forma_de_pagamento"`
}
