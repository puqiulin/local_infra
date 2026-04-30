#!/bin/sh
cat > /etc/seaweedfs/s3.json <<EOF
{
  "identities": [
    {
      "name": "admin",
      "credentials": [
        {
          "accessKey": "${SEAWEEDFS_S3_ACCESS_KEY}",
          "secretKey": "${SEAWEEDFS_S3_SECRET_KEY}"
        }
      ],
      "actions": [
        "Admin",
        "Read",
        "List",
        "Tagging",
        "Write"
      ]
    }
  ]
}
EOF

exec weed filer -master=seaweedfs-master:9333 -port=8888 -s3 -s3.port=8333 -s3.config=/etc/seaweedfs/s3.json -metricsPort=9326
