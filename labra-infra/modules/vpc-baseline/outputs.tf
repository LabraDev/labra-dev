output "vpc_id" {
  value = aws_vpc.baseline.id
}

output "public_subnet_ids" {
  value = [for s in aws_subnet.public : s.id]
}

output "private_subnet_ids" {
  value = [for s in aws_subnet.private : s.id]
}

output "internet_gateway_id" {
  value = aws_internet_gateway.baseline.id
}

output "nat_gateway_id" {
  value = var.enable_nat_gateway ? aws_nat_gateway.baseline[0].id : null
}
