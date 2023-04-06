# Dad Joke Generator 🤣

This project is a Dad Joke Generator that uses the OpenAI GPT-3 API to generate the funniest dad jokes you've ever heard! The application is built using Go and is composed of a joke-server, joke-worker, Redis, MongoDB, and NATS.

## Table of Contents 📚

- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Deployment](#deployment)
- [License](#license)

## Architecture 🏗️
The application consists of the following components:

1. **joke-server**: A web server that listens for incoming HTTP requests and returns a dad joke.
2. **joke-worker**: A background worker that communicates with the OpenAI GPT-3 API, generates dad jokes, caches them in Redis, and stores them in MongoDB.
3. **Redis**: A caching layer that temporarily stores generated dad jokes.
4. **MongoDB**: A NoSQL database that permanently stores generated dad jokes.
5. **NATS**: A messaging system that facilitates communication between the joke-server and joke-worker.

<!-- ![Application Architecture](architecture.png) -->

## Prerequisites ⚙️

Before you can deploy the Dad Joke Generator, you'll need the following:

- Docker and Docker Compose installed on your machine
- An OpenAI API key
- A GitHub account to clone the repository

## Deployment 🚀

To deploy the Dad Joke Generator, follow these steps:

1. Clone the repository:

```bash
git clone https://github.com/vfiftyfive/dadjokes.git
cd dadjokes/docker
```

2. Create a .env file in the dadjokes/docker directory, with your OpenAI API key:
```bash
echo "OPENAI_API_KEY=your_api_key_here" > .env
```

3. Run the deployment script:
```bash
make deploy
```

4. Generate a lot of jokes:
```bash 
for i in {1..30}; do curl http://localhost:8080/joke; echo -e; done
```

## License 📄
This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.