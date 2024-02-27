# go-email-validator
Cheap &amp; Cheerful E-Mail address validation in go

# Basic Usage

```go
email := "test@email.com"

res, err := emailvalidator.BuildResult(email)
if err != nil {
    panic(fmt.Sprintf("email validation failed: %v", err))
}

fmt.Println(res)
```