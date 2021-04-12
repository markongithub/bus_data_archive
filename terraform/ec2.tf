resource "aws_instance" "first_db" {
  ami           = "ami-0ca5c3bd5a268e7db"
  instance_type = "t2.nano"
  iam_instance_profile         = "my_first_ec2_role"
}

data "aws_iam_policy_document" "instance-assume-role-policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "ec2_role" {
  name = "my_first_ec2_role"
  assume_role_policy = data.aws_iam_policy_document.instance-assume-role-policy.json
}

resource "aws_s3_bucket_policy" "ec2_s3" {
  bucket = aws_s3_bucket.first_bucket.id

  policy = jsonencode({
    Version = "2012-10-17"
    Id      = "what_is_this_id_for"
    Statement = [
      {
        Sid       = "root_on_that_one_bucket"
        Effect    = "Allow"
        Principal = {
          AWS = [aws_iam_role.ec2_role.arn]
        }
        Action    = "s3:*"
        Resource = [
          aws_s3_bucket.first_bucket.arn,
          "${aws_s3_bucket.first_bucket.arn}/*",
        ]
      },
    ]
  })
}
