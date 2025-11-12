# Upload Drive Script

Sistema backend em Go para envio de arquivos para o Google Drive. Suporta:

* Upload via **form-data** (`/upload`)
* Upload via **URL** (`/upload-url`)
* OAuth2 automático via navegador

---

## 🏗️ Estrutura do projeto

```
upload-drive-script/
├── go.mod
├── go.sum
├── credentials.json        # Credenciais OAuth2 do Google
├── cmd/
│   ├── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── handlers/
│   │   └── drive_handler.go
│   ├── services/
│   │   └── drive_service.go
├── pkg/
│   ├── logger/
│   │   └── logger.go
```

---

## ⚙️ Requisitos

* Go 1.20+
* Conta Google com OAuth2 credentials (JSON)
* Variáveis de ambiente opcionais para customização (ver seção Configuração)
* `ffmpeg` instalado no host (necessário para extrair o áudio dos vídeos)

---

## 💻 Build e execução

### Ambiente local (Go)

1. Clone o repositório:

```bash
git clone https://github.com/GustavoGebhardt/upload-drive-script
cd upload-drive-script
```

2. Instale dependências:

```bash
go mod tidy
```

3. Build do projeto:

```bash
go build -o upload-drive-script ./cmd
```

4. Execute (ajuste os valores conforme o ambiente):

```bash
HTTP_LISTEN_ADDR=:3000 ./upload-drive-script
```

O servidor vai iniciar em `http://localhost:3000`.

### Utilizando Docker

1. Construa a imagem localmente:

```bash
docker build -t upload-drive-script .
```

2. Crie um arquivo `.env` com as variáveis necessárias (ajuste conforme seu ambiente):

```bash
APP_SERVER_PORT=:3000
GOOGLE_CREDENTIALS_FILE=credentials.json
GOOGLE_TOKEN_FILE=token.json
```

3. Execute o container em segundo plano expondo a porta 3000, carregando o `.env` e montando as credenciais:

```bash
docker run -d -p 3000:3000 \
  --name upload-drive-script \
  --env-file .env \
  upload-drive-script
```

4. Ajuste variáveis ou mounts conforme necessário (por exemplo `GOOGLE_TOKEN_FILE`, `APP_SERVER_PORT` ou outro caminho de credenciais). O serviço seguirá disponível em `http://localhost:3000`. Pare o container com `docker stop upload-drive-script`.

---

## 🔧 Configuração via variáveis de ambiente

| Variável                  | Descrição                                            | Padrão                               |
|---------------------------|------------------------------------------------------|--------------------------------------|
| `GOOGLE_CREDENTIALS_FILE` | Caminho para o JSON de credenciais OAuth             | `credentials.json`                   |
| `GOOGLE_AUTH_MODE`        | Tipo de autenticação: `oauth` ou `service_account`   | `oauth`                              |
| `GOOGLE_TOKEN_FILE`       | Caminho onde o token OAuth autorizado será persistido | `token.json`                         |
| `APP_BASE_URL`            | URL base da aplicação                                | `localhost`                          |
| `GOOGLE_OAUTH_STATE`      | Valor de state usado na autorização OAuth            | `state-token`                        |
| `APP_SERVER_PORT`         | Endereço/porta que o servidor HTTP deve escutar      | `:3000`                              |

Defina as variáveis antes de executar o binário:

```bash
export GOOGLE_CREDENTIALS_FILE=./secrets/credentials.json
export HTTP_LISTEN_ADDR=:8080
./upload-drive-script
```

---

## 🔑 Autenticação Google Drive

> Este fluxo é necessário apenas quando `GOOGLE_AUTH_MODE=oauth`. Ao usar `service_account`, não há etapa manual de autorização.

1. Com o navegador acesse a rota `/auth`:

```
http://localhost:3000/auth
```

2. Você será redirecionado para a página de login do Google.
3. Após autorizar, a callback `/oauth2callback` salvará o token em `token.json`.

---

## 📤 Rotas

### 1. Upload via form-data

**POST** `/upload`

**Body:** `form-data`

| Campo       | Descrição                       |
| ----------- | ------------------------------- |
| `file`      | Arquivo a ser enviado           |
| `folder_id` | (Opcional) ID da pasta no Drive |
| `file_name` | (Opcional) Nome do arquivo no Drive |

**Exemplo curl:**

```bash
curl -X POST http://localhost:3000/upload \
  -F "file=@/caminho/para/arquivo.mp3" \
  -F "folder_id=ID_DA_PASTA" \
  -F "file_name=novo-nome.mp3"
```

Se o arquivo for um vídeo, o backend extrai o áudio automaticamente e envia ao Drive, retornando também `audio_file_id` na resposta.

---

### 2. Upload via URL

**POST** `/upload-url`

**Body:** `form-data`

| Campo       | Descrição                       |
| ----------- | ------------------------------- |
| `url`       | URL pública do arquivo          |
| `folder_id` | (Opcional) ID da pasta no Drive |
| `file_name` | (Opcional) Nome do arquivo no Drive |

**Exemplo curl:**

```bash
curl -X POST http://localhost:3000/upload-url \
  -d "url=https://example.com/audio.mp3" \
  -d "folder_id=ID_DA_PASTA" \
  -d "file_name=novo-nome.mp3"
```

Uploads de vídeo via URL seguem o mesmo fluxo: o arquivo de vídeo é enviado e o áudio extraído é enviado separadamente, retornando `audio_file_id` quando aplicável.

---

## ⚡ Observações

* Para arquivos muito grandes (>1GB), o upload é **resumable** e dividido em chunks de 10MB.
* Tokens OAuth2 são salvos no arquivo definido por `GOOGLE_TOKEN_FILE`; mantenha-o fora do controle de versão.
* Defina `GOOGLE_AUTH_MODE=service_account` para usar uma Service Account; nesse modo as rotas `/auth` e `/oauth2callback` não ficam disponíveis e o arquivo definido em `GOOGLE_CREDENTIALS_FILE` deve conter o JSON da Service Account.
