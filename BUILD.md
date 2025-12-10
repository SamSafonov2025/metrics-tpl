# Инструкции по сборке

Этот документ описывает, как собрать приложения с информацией о версии сборки.

## Информация о сборке

При старте приложения выводится информация о сборке:

```
Build version: v1.0.0
Build date: 2025/11/30 17:28:27
Build commit: abc123def
```

Если информация не была указана при сборке, выводится `N/A`:

```
Build version: N/A
Build date: N/A
Build commit: N/A
```

## Сборка с ldflags

### Linux/macOS (Bash)

#### Server

```bash
go build \
  -ldflags "-X 'main.buildVersion=v1.0.0' \
            -X 'main.buildDate=$(date +'%Y/%m/%d %H:%M:%S')' \
            -X 'main.buildCommit=$(git rev-parse --short HEAD)'" \
  -o bin/server \
  ./cmd/server
```

#### Agent

```bash
go build \
  -ldflags "-X 'main.buildVersion=v1.0.0' \
            -X 'main.buildDate=$(date +'%Y/%m/%d %H:%M:%S')' \
            -X 'main.buildCommit=$(git rev-parse --short HEAD)'" \
  -o bin/agent \
  ./cmd/agent
```

### Windows (PowerShell)

**Рекомендуемый способ - используйте готовый скрипт `build.ps1`:**

```powershell
# Собрать server и agent
.\build.ps1

# Собрать только server
.\build.ps1 -Target server

# Собрать с кастомной версией
.\build.ps1 -Version "v1.0.0"
```

**Или вручную:**

#### Server

```powershell
$VERSION = "v1.0.0"
$BUILD_DATE = Get-Date -Format "yyyy/MM/dd HH:mm:ss"
$GIT_COMMIT = git rev-parse --short HEAD

go build `
  -ldflags "-X 'main.buildVersion=$VERSION' -X 'main.buildDate=$BUILD_DATE' -X 'main.buildCommit=$GIT_COMMIT'" `
  -o bin/server.exe `
  ./cmd/server
```

#### Agent

```powershell
$VERSION = "v1.0.0"
$BUILD_DATE = Get-Date -Format "yyyy/MM/dd HH:mm:ss"
$GIT_COMMIT = git rev-parse --short HEAD

go build `
  -ldflags "-X 'main.buildVersion=$VERSION' -X 'main.buildDate=$BUILD_DATE' -X 'main.buildCommit=$GIT_COMMIT'" `
  -o bin/agent.exe `
  ./cmd/agent
```

### Windows (CMD)

Для CMD создайте скрипт `build.bat`:

```batch
@echo off
set VERSION=v1.0.0
for /f %%i in ('git rev-parse --short HEAD') do set GIT_COMMIT=%%i

go build -ldflags "-X main.buildVersion=%VERSION% -X main.buildCommit=%GIT_COMMIT%" -o bin\server.exe ./cmd/server
```

**Примечание:** В CMD сложно получить форматированную дату, поэтому рекомендуется использовать PowerShell.

## Запуск через go run

### Linux/macOS (Bash)

#### Server

```bash
go run \
  -ldflags "-X 'main.buildVersion=dev' \
            -X 'main.buildDate=$(date +'%Y/%m/%d %H:%M:%S')' \
            -X 'main.buildCommit=local'" \
  ./cmd/server
```

#### Agent

```bash
go run \
  -ldflags "-X 'main.buildVersion=dev' \
            -X 'main.buildDate=$(date +'%Y/%m/%d %H:%M:%S')' \
            -X 'main.buildCommit=local'" \
  ./cmd/agent
```

### Windows (PowerShell)

#### Server

```powershell
$BUILD_DATE = Get-Date -Format "yyyy/MM/dd HH:mm:ss"

go run `
  -ldflags "-X 'main.buildVersion=dev' -X 'main.buildDate=$BUILD_DATE' -X 'main.buildCommit=local'" `
  ./cmd/server
```

#### Agent

```powershell
$BUILD_DATE = Get-Date -Format "yyyy/MM/dd HH:mm:ss"

go run `
  -ldflags "-X 'main.buildVersion=dev' -X 'main.buildDate=$BUILD_DATE' -X 'main.buildCommit=local'" `
  ./cmd/agent
```

## Сборка без версии

Если собрать без ldflags, будут использованы значения по умолчанию (`N/A`):

```bash
go build -o bin/server ./cmd/server
go build -o bin/agent ./cmd/agent
```

## Переменные

В коде определены следующие переменные:

```go
var (
    buildVersion string = "N/A"
    buildDate    string = "N/A"
    buildCommit  string = "N/A"
)
```

Они могут быть переопределены через `-ldflags` при сборке:

- `main.buildVersion` - версия приложения
- `main.buildDate` - дата и время сборки
- `main.buildCommit` - git commit hash

## Примеры для разных окружений

### Development

**Linux/macOS:**
```bash
go build \
  -ldflags "-X main.buildVersion=dev" \
  -o bin/server \
  ./cmd/server
```

**Windows (PowerShell):**
```powershell
go build -ldflags "-X main.buildVersion=dev" -o bin/server.exe ./cmd/server
```

### Staging

**Linux/macOS:**
```bash
go build \
  -ldflags "-X main.buildVersion=staging-$(date +%Y%m%d)" \
  -o bin/server \
  ./cmd/server
```

**Windows (PowerShell):**
```powershell
$DATE_SUFFIX = Get-Date -Format "yyyyMMdd"
go build -ldflags "-X main.buildVersion=staging-$DATE_SUFFIX" -o bin/server.exe ./cmd/server
```

### Production

**Linux/macOS:**
```bash
VERSION=$(git describe --tags --exact-match 2>/dev/null || echo "unknown")
BUILD_DATE=$(date +'%Y/%m/%d %H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD)

go build \
  -ldflags "-X 'main.buildVersion=$VERSION' \
            -X 'main.buildDate=$BUILD_DATE' \
            -X 'main.buildCommit=$GIT_COMMIT'" \
  -o bin/server \
  ./cmd/server
```

**Windows (PowerShell):**
```powershell
$VERSION = git describe --tags --exact-match 2>$null
if (-not $VERSION) { $VERSION = "unknown" }
$BUILD_DATE = Get-Date -Format "yyyy/MM/dd HH:mm:ss"
$GIT_COMMIT = git rev-parse --short HEAD

go build `
  -ldflags "-X 'main.buildVersion=$VERSION' -X 'main.buildDate=$BUILD_DATE' -X 'main.buildCommit=$GIT_COMMIT'" `
  -o bin/server.exe `
  ./cmd/server
```

## CI/CD интеграция

### GitHub Actions

```yaml
- name: Build with version
  run: |
    VERSION=${{ github.ref_name }}
    BUILD_DATE=$(date +'%Y/%m/%d %H:%M:%S')
    GIT_COMMIT=${{ github.sha }}

    go build \
      -ldflags "-X 'main.buildVersion=$VERSION' \
                -X 'main.buildDate=$BUILD_DATE' \
                -X 'main.buildCommit=$GIT_COMMIT'" \
      -o bin/server \
      ./cmd/server
```

### GitLab CI

```yaml
build:
  script:
    - VERSION=${CI_COMMIT_TAG:-${CI_COMMIT_SHORT_SHA}}
    - BUILD_DATE=$(date +'%Y/%m/%d %H:%M:%S')
    - GIT_COMMIT=${CI_COMMIT_SHORT_SHA}
    - |
      go build \
        -ldflags "-X 'main.buildVersion=$VERSION' \
                  -X 'main.buildDate=$BUILD_DATE' \
                  -X 'main.buildCommit=$GIT_COMMIT'" \
        -o bin/server \
        ./cmd/server
```

## Проверка версии

После сборки проверьте версию:

```bash
./bin/server &
# Посмотрите первые строки вывода

# или
timeout 1 ./bin/server 2>&1 | head -3
```

Вывод должен содержать:

```
Build version: v1.0.0
Build date: 2025/11/30 17:28:27
Build commit: abc123d
```

## Troubleshooting

### Git команды не работают

Убедитесь, что находитесь в git репозитории:

```bash
git status
```

Если нет, инициализируйте:

```bash
git init
git add .
git commit -m "Initial commit"
git tag v1.0.0
```

### Важные замечания

**Кавычки в ldflags:**
- Linux/macOS: используйте одинарные кавычки для значений с пробелами
  ```bash
  -ldflags "-X 'main.buildDate=$(date +'%Y/%m/%d %H:%M:%S')'"
  ```

- Windows (PowerShell): используйте переменные для значений с пробелами
  ```powershell
  $BUILD_DATE = Get-Date -Format "yyyy/MM/dd HH:mm:ss"
  -ldflags "-X 'main.buildDate=$BUILD_DATE'"
  ```

**Перенос строк:**
- Linux/macOS: используйте обратный слеш `\`
- PowerShell: используйте обратный апостроф `` ` ``

**Расширения файлов:**
- Linux/macOS: без расширения (`bin/server`)
- Windows: с расширением `.exe` (`bin/server.exe`)
