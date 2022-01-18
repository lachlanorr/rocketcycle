{
  "name": "rpg",
  "environment": "dev",
  "concerns": [
    {
      "type": "GENERAL",
      "name": "edge",

      "topics": [
        {
          "name": "response",
          "current": {
            "partitionCount": 1,
            "cluster": "${kafka_cluster}"
          },
          "consumerPrograms": [
            {
              "name": "./@platform",
              "args": ["edge", "serve",
                       "-t", "@topic",
                       "-p", "@partition",
                       "-e", "@environment",
                       "--edge",
                       "--admin_brokers", "@admin_brokers",
                       "--consumer_brokers", "@consumer_brokers",
                       "--http_addr", ":1135@partition",
                       "--grpc_addr", ":1136@partition",
                       "--stream", "@stream",
                       "--otelcol_endpoint", "@otelcol_endpoint"
                      ],
              "abbrev": "edge/@partition",
              "tags": {"service.name": "rkcy.@platform.@environment.@concern"}
            }
          ]
        }
      ]
    },
    {
      "type": "APECS",
      "name": "Player",

      "topics": [
        {
          "name": "process",
          "current": {
            "partitionCount": 1,
            "cluster": "${kafka_cluster}"
          }
        },
        {
          "name": "storage",
          "current": {
            "partitionCount": 1,
            "cluster": "${kafka_cluster}"
          }
        }
      ]
    },
    {
      "type": "APECS",
      "name": "Character",

      "topics": [
        {
          "name": "process",
          "current": {
            "partitionCount": 1,
            "cluster": "${kafka_cluster}"
          }
        },
        {
          "name": "storage",
          "current": {
            "partitionCount": 1,
            "cluster": "${kafka_cluster}"
          }
        }
      ]
    }
  ],

  "clusters": [
    {
      "name": "${kafka_cluster}",
      "brokers": "${join(":9092,", kafka_hosts)}:9092",
      "isAdmin": true
    }
  ],

  "storageTargets": [
    {
      "name": "pg-0",
      "type": "postgresql",
      "isPrimary": true,
      "config": {
        "connString": "postgresql://postgres@${postgresql_hosts[0]}:5432/rpg"
      }
    }
  ]
}
