# Sales assistant

Software assistente de vendas com OpenAI.

## Parâmetros de inicialização

Para inicializar pelo VSCode, ver arquivo _.vscode/launch.example.json_

```
amqp        => URL do RabbitMQ, ex. amqp://admin:admin@localhost:5672
exchange    => Nome da exchange definido pelo Evolution API
instanceid  => ID da instância no Evolution API
evotoken    => Token da instância do Evolution API
evourl      => URL da API do EvolutionAPI, ex. http://localhost:9339
openaitoken => Token do projeto da OpenAI
pg          => URL de conexão postgres, ex. postgresql://postgres:secret@127.0.0.1:5432/talkassist?search_path=maromba
number      => Número do celular com código do país e DDD de quem receberá as propostas resolvidas pela IA, ex. 5599123456789
me          => Ignorar mensagens enviadas por mim mesmo
```
