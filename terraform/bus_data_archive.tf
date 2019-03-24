provider "aws" {
  region = "us-west-2"
}

resource "aws_s3_bucket" "first_bucket" {
  bucket = "busdata-00-us-west-2"
  acl    = "public-read"

  website {
    index_document = "index.html"
    error_document = "error.html"

  }

  policy = <<POLICY
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "PublicReadGetObject",
            "Effect": "Allow",
            "Principal": "*",
            "Action": [
                "s3:GetObject"
            ],
            "Resource": [
                "arn:aws:s3:::busdata-00-us-west-2/*"
            ]
        }
    ]
}
POLICY

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET"]
    allowed_origins = ["*"]
  }
}

