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

resource "aws_db_instance" "busdata00" {
  storage_type            = "gp2"
  engine                  = "postgres"
  engine_version          = "10.6"
  instance_class          = "db.t2.micro"
  license_model           = "postgresql-license"
  allocated_storage       = "20"
  availability_zone       = "us-west-2b"
  backup_retention_period = "7"
  backup_window           = "11:23-11:53"
  copy_tags_to_snapshot   = "true"
  db_subnet_group_name    = "default"
  deletion_protection     = "true"
  maintenance_window      = "tue:08:26-tue:08:56"
  monitoring_interval     = "60"
  monitoring_role_arn     = "arn:aws:iam::935801117323:role/rds-monitoring-role"
  multi_az                = "false"
  name                    = "busdata00"
  option_group_name       = "default:postgres-10"
  parameter_group_name    = "default.postgres10"
  port                    = "5432"
  publicly_accessible     = "true"
  skip_final_snapshot     = "true"

  tags = {
    workload-type = "other"
  }

  username = "busdata_user"
}
