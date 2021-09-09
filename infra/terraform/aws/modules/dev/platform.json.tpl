{
    "name": "rpg",
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
                    "consumerProgram": {
                        "name": "./@platform",
                        "args": ["edge", "serve", "--admin_brokers", "@admin_brokers", "--consumer_brokers", "@consumer_brokers", "-t", "@topic", "-p", "@partition", "--http_addr", ":1135@partition", "--grpc_addr", ":1136@partition"],
                        "abbrev": "edge/@partition",
                        "tags": {"service.name": "rkcy.@platform.@concern"}
                    }
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
            "brokers": "${join(":9092,", kafka_hosts)}:9092"
        }
    ]
}
