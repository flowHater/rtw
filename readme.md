# How to use it

## install

`go install github.com/flowHater/rtw@v0.0.1`

## declare all your command/service in a services.json

```json
{
  "rootPath": "/home/myuser/Src/",
  "services": [
    {
      "path": "service-a",
      "command": "npm",
      "args": ["run", "start"],
      "name": "service-a:api"
    },
    {
      "path": "service-a",
      "command": "npm",
      "args": ["run", "start:worker1"],
      "name": "service-a:worker1"
    },
    {
      "path": "service-b",
      "command": "npm",
      "args": ["run", "start:api"],
      "name": "service-b:api"
    },
    {
      "path": "service-b",
      "command": "go",
      "args": ["run", "./cmd/worker37"],
      "name": "service-b:worker37"
    },
    {
      "path": "front",
      "command": "yarn",
      "args": ["start"],
      "name": "front"
    }
  ]
}
```

## run

`rtw --file service.json [--stopOnError]`
Because rtw will prefix each log line by the service name, you can redirect the command to output in a log file. `tail -f logs | grep "servicename"` to watch only a specific service.
