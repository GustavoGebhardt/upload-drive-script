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

4. Execute:

```bash
./upload-drive-script
```

O servidor vai iniciar em `http://localhost:3000`.

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

* Para arquivos muito grandes (>1GB), o upload é **resumable** e dividido em chunks de 10MB
* Tokens OAuth2 são salvos em `token.json` para reutilização
