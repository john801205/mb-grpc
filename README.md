# mb-grpc

This repository is a mountebank plugin for the gRPC protocol.

## Features

1. Support both unary and streaming RPCs
2. Support gRPC metadata including header and trailer
3. Support gRPC status including code, message and details
4. Support mountebank proxy mode
5. Support gRPC reflection server

## Getting Started

### Dependencies

* mountebank
* go
* protoc

All should be searchable through the PATH environment variable.

### Build

```shell
$ make build
```

This will produce a binary program `mb-grpc` under the bin directory.


### Run

After the plugin is built, you can run it through the mountebank program.

Create protocols.json file for gRPC:

```json
{
  "grpc": {
    "createCommand": "bin/mb-grpc"
  }
}
```

Create imposters.json file:

```json
{
  "imposters": [
    {
      "port": 5568,
      "protocol": "grpc",
      "recordRequests": true,
      "options": {
        "protoc": {
          "importDirs": [
            "./"
          ],
          "protoFiles": [
            "./service.proto"
          ]
        }
      }
    }
  ]
}
```

Start Mountebank with protocols file:

```shell
$ mb start --port 2525 --protofile protocols.json --configfile imposters.json --loglevel info
```

### Examples

For unary RPCs,

```json
{
  "responses": [
    {
      "is": {
        "header": {
          "aaaa": [
            "bbb"
          ],
          "mb-grpc-data-bin": [
            "44GE44Gh44Gw44KT"
          ]
        },
        "message": {
          "pong": "Hello, world"
        },
        "trailer": {
          "bbbb": [
            "aaa"
          ],
          "mb-grpc-data-bin": [
            "44GE44Gh44Gw44KT"
          ]
        },
        "status": {
          "code": "OK"
        }
      }
    }
  ],
  "predicates": [
    {
      "equals": {
        "method": "/pingpong.Service/PingPong",
        "messages": [
          {
            "from": "client",
            "message": {
              "ping": "success"
            }
          }
        ]
      }
    }
  ]
}
```

For streaming RPCs, there are a sequence of messages from either client or server.

```json
{
  "responses": [
    {
      "is": {
        "header": {
          "symbol": [
            "-_.~!#$&'()*+,/:;=?@[]%20"
          ],
          "mb-grpc-data-bin": [
            "44GE44Gh44Gw44KT"
          ]
        },
        "message": {
          "pong": "Hello, world"
        }
      }
    },
    {
      "is": {
        "message": {
          "pong": "Hello, world"
        }
      },
      "repeat": 2
    },
    {
      "is": {
        "message": {
          "pong": "Hello, world"
        },
        "trailer": {
          "symbol": [
            "-_.~!#$&'()*+,/:;=?@[]%20"
          ],
          "mb-grpc-data-bin": [
            "44GE44Gh44Gw44KT"
          ]
        },
        "status": {
          "code": "ABORTED",
          "message": "message",
          "details": [
            {
              "type": "pingpong.Pong",
              "message": {
                "pong": "Hello, world"
              }
            }
          ]
        }
      }
    }
  ],
  "predicates": [
    {
      "equals": {
        "method": "/pingpong.Service/PingPingPongPong",
        "requestHeader": {
          "flag": [
            "failure - pongpingpingpongpongpong"
          ]
        }
      }
    },
    {
      "or": [
        {
          "deepEquals": {
            "messages": []
          }
        },
        {
          "deepEquals": {
            "messages": [
              {
                "from": "server",
                "message": {
                  "pong": "Hello, world"
                }
              },
              {
                "from": "client",
                "message": {
                  "ping": "failure"
                }
              },
              {
                "from": "client",
                "message": {
                  "ping": "failure"
                }
              }
            ]
          }
        },
        {
          "deepEquals": {
            "messages": [
              {
                "from": "server",
                "message": {
                  "pong": "Hello, world"
                }
              },
              {
                "from": "client",
                "message": {
                  "ping": "failure"
                }
              },
              {
                "from": "client",
                "message": {
                  "ping": "failure"
                }
              },
              {
                "from": "server",
                "message": {
                  "pong": "Hello, world"
                }
              }
            ]
          }
        },
        {
          "deepEquals": {
            "messages": [
              {
                "from": "server",
                "message": {
                  "pong": "Hello, world"
                }
              },
              {
                "from": "client",
                "message": {
                  "ping": "failure"
                }
              },
              {
                "from": "client",
                "message": {
                  "ping": "failure"
                }
              },
              {
                "from": "server",
                "message": {
                  "pong": "Hello, world"
                }
              },
              {
                "from": "server",
                "message": {
                  "pong": "Hello, world"
                }
              }
            ]
          }
        }
      ]
    }
  ]
}
```

It is also viable to proxy to other gRPC server.
```json
{
  "responses": [
    {
      "proxy": {
        "to": "localhost:5569",
        "mode": "proxyOnce",
        "predicateGenerators": [
          {
            "matches": {
              "method": true,
              "messages": true,
              "requestHeader": true,
              "responseHeader": true
            }
          }
        ]
      }
    }
  ],
  "predicates": [
    {
      "exists": {
        "requestHeader": {
          "proxy-mode": true
        }
      }
    }
  ]
}
```
