# Black and White Image Converter Microservice

This Go project sets up an HTTP microservice that accepts an image file via a POST request, assigns it an ID, converts it to black and white, and saves it in a temporary directory. The client can then send metadata along with the ID, and the server will re-associate the file and the metadata and return the resulting image.

## Prerequisites

- Go 1.23 or later

## 

1. **Build and push to docker:**

    ```sh
    make all
    ```

2. **Install dependencies:**

    ```sh
    go get github.com/disintegration/imaging
    go get github.com/google/uuid
    ```

3. **Build and run the server locally:**

    ```sh
    go build -o blackandwhite main.go
    ./blackandwhite
    ```

    Optionally, you can set the `TEMP_DIR` environment variable to specify a different directory for storing temporary images:

    ```sh
    export TEMP_DIR=/path/to/temp/dir
    ./blackandwhite
    ```

4. **Send HTTP requests to the server:**

    The server listens on `:8080`. You can use `curl` or any HTTP client to send requests.

### Upload an Image

```sh
curl -X POST -F "file=@/path/to/your/image.jpg" http://localhost:8080/image
```

Response:

```json
{
  "id": "your-unique-image-id"
}
```

### Send Metadata and Retrieve the Image

```sh
curl -X GET -o output.jpg  http://localhost:8080/image?id=your-unique-image-id
```

The response will be the black and white image.

## Authorization
