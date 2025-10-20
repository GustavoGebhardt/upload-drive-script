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

---

## 💻 Build e execução

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

---

## 🔧 Configuração via variáveis de ambiente

| Variável                     | Descrição                                               | Padrão                               |
|------------------------------|---------------------------------------------------------|--------------------------------------|
| `GOOGLE_CREDENTIALS_FILE`    | Caminho para o JSON de credenciais OAuth                | `credentials.json`                   |
| `GOOGLE_TOKEN_FILE`          | Caminho onde o token OAuth autorizado será persistido   | `token.json`                         |
| `GOOGLE_OAUTH_REDIRECT_URL`  | URL callback registrada no console Google               | `http://localhost:3000/oauth2callback` |
| `GOOGLE_OAUTH_STATE`         | Valor de state usado na autorização OAuth               | `state-token`                        |
| `HTTP_LISTEN_ADDR`           | Endereço/porta que o servidor HTTP deve escutar         | `:3000`                              |

Defina as variáveis antes de executar o binário:

```bash
export GOOGLE_CREDENTIALS_FILE=./secrets/credentials.json
export HTTP_LISTEN_ADDR=:8080
./upload-drive-script
```

---

## 🔑 Autenticação Google Drive

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

**Exemplo curl:**

```bash
curl -X POST http://localhost:3000/upload \
  -F "file=@/caminho/para/arquivo.mp3" \
  -F "folder_id=ID_DA_PASTA"
```

---

### 2. Upload via URL

**POST** `/upload-url`

**Body:** `form-data`

| Campo       | Descrição                       |
| ----------- | ------------------------------- |
| `url`       | URL pública do arquivo          |
| `folder_id` | (Opcional) ID da pasta no Drive |

**Exemplo curl:**

```bash
curl -X POST http://localhost:3000/upload-url \
  -d "url=https://example.com/audio.mp3" \
  -d "folder_id=ID_DA_PASTA"
```

---

## ⚡ Observações

* Para arquivos muito grandes (>1GB), o upload é **resumable** e dividido em chunks de 10MB.
* Tokens OAuth2 são salvos no arquivo definido por `GOOGLE_TOKEN_FILE`; mantenha-o fora do controle de versão.
