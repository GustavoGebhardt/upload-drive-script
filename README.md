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

O serviço agora opera de forma **stateless** em relação à autenticação do usuário para uploads. O token de acesso OAuth2 deve ser obtido pelo cliente (frontend) e passado para este serviço.

### Fluxo de Autenticação (Centralizado no Frontend)

1.  O Frontend (`client-post-forge`) realiza o login do usuário com o Google e obtém o escopo `https://www.googleapis.com/auth/drive.file`.
2.  O Frontend envia o arquivo para este serviço (`/upload` ou `/upload-url`) incluindo o **Access Token** no cabeçalho.
3.  Este serviço utiliza o token recebido para autenticar diretamente com a API do Google Drive e realizar o upload na conta do usuário.

> **Nota:** As rotas `/auth` e `/oauth2callback` ainda existem para fluxos legados ou de manutenção, mas não são utilizadas para o fluxo principal de upload de usuários.

---

## 📤 Rotas

Todas as rotas de upload esperam o cabeçalho de autorização:

```http
Authorization: Bearer <GOOGLE_ACCESS_TOKEN>
```

### 1. Upload via form-data

**POST** `/upload`

**Headers:**
*   `Authorization: Bearer <seu_token_de_acesso>`

**Body:** `form-data`

| Campo       | Descrição                                         |
| ----------- | ------------------------------------------------- |
| `file`      | Arquivo a ser enviado (**somente áudio/vídeo**)    |
| `folder_id` | (Opcional) ID da pasta no Drive                   |
| `file_name` | (Opcional) Nome do arquivo no Drive               |

**Exemplo curl:**

```bash
curl -X POST http://localhost:3000/upload \
  -H "Authorization: Bearer ya29.a0..." \
  -F "file=@/caminho/para/arquivo.mp3" \
  -F "folder_id=ID_DA_PASTA" \
  -F "file_name=novo-nome.mp3"
```

Exemplos de retorno:

**Vídeo (com áudio extraído):**

```json
{
  "video_file_id": "1f9VOBVoDDc1jb6menibyU0PmPx4xUX5R",
  "audio_file_id": "18eXy3meiR22pXyZ7ygqjxRWTInHaureR",
  "video_file_url": "https://upload-script.clientpostforge.com/uploads/video.mp4",
  "audio_file_url": "https://upload-script.clientpostforge.com/uploads/video-audio.mp3"
}
```

**Áudio:**

```json
{
  "video_file_id": null,
  "audio_file_id": "18eXy3meiR22pXyZ7ygqjxRWTInHaureR",
  "video_file_url": null,
  "audio_file_url": "https://upload-script.clientpostforge.com/uploads/audio.mp3"
}
```

`video_file_url` e `audio_file_url` apontam para cópias locais expostas em `/uploads/<arquivo>`.

---

### 2. Upload via URL

**POST** `/upload-url`

**Headers:**
*   `Authorization: Bearer <seu_token_de_acesso>`

**Body:** `form-data`

| Campo       | Descrição                                         |
| ----------- | ------------------------------------------------- |
| `url`       | URL pública do arquivo (**somente áudio/vídeo**)   |
| `folder_id` | (Opcional) ID da pasta no Drive                   |
| `file_name` | (Opcional) Nome do arquivo no Drive               |

**Exemplo curl:**

```bash
curl -X POST http://localhost:3000/upload-url \
  -H "Authorization: Bearer ya29.a0..." \
  -d "url=https://example.com/audio.mp3" \
  -d "folder_id=ID_DA_PASTA" \
  -d "file_name=novo-nome.mp3"
```

Uploads via URL retornam o mesmo payload mostrado na rota `/upload`. O serviço baixa o arquivo, o replica em `/uploads` e extrai o áudio sempre que o MIME indicar vídeo.

---

## ⚡ Observações

* **Token Obrigatório:** O token de acesso é mandatório para autenticar o upload na conta do usuário correto.
* Apenas arquivos com MIME `audio/*` ou `video/*` são aceitos; qualquer outro tipo retorna HTTP 400.
* Para arquivos muito grandes (>1GB), o upload é **resumable** e dividido em chunks de 10MB.
* Tokens OAuth2 salvos localmente (`token.json`) são ignorados quando o header `Authorization` é fornecido.
* Defina `GOOGLE_AUTH_MODE=service_account` para usar uma Service Account como fallback global, mas o token do usuário sempre terá prioridade se fornecido.
