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

![image](https://user-images.githubusercontent.com/7715763/232326149-3461b3c6-346b-4cbd-95f5-774587464342.png)


## Prerequisites ⚙️

Before you can deploy the Dad Joke Generator, you'll need the following:

- Docker and Docker Compose installed on your machine
- An OpenAI API key
- Access to a Kubernetes cluster, `helm` and `kubectl` installed if you want to deploy the application on Kubernetes
- `gpg` command-line available

## Deployment 🚀

### Docker Compose (Local) 🐳 

To deploy the Dad Joke Generator, follow these steps:

1. Clone the repository:

```bash
git clone https://github.com/vfiftyfive/dadjokes.git
cd dadjokes/deploy/docker
```

2. Create a .env file in the `dadjokes/deploy/docker` directory, with your OpenAI API key:
```bash
echo "OPENAI_API_KEY=your_api_key_here" > .env
```

3. Run the deployment script:
```bash
cd ../..
make deploy
```

4. Generate a lot of jokes!
```bash
for i in {1..30}; do curl http://localhost:8080/joke; echo -e; done
```
The last 10 jokes should come a lot faster than the first 20, as the joke-worker will retrieve jokes from the Redis cache after that, for a time defined in `constants.RedisTTL`.

### Kubernetes ☸

1. Clone the repository and change the directory to `deploy/devspace`:

```bash
git clone https://github.com/vfiftyfive/dadjokes.git
cd dadjokes/deploy/devspace
```

2. Install DevSpace:
```bash 
# AMD64
curl -L -o devspace "https://github.com/loft-sh/devspace/releases/latest/download/devspace-linux-amd64" && sudo install -c -m 0755 devspace /usr/local/bin

# ARM64
curl -L -o devspace "https://github.com/loft-sh/devspace/releases/latest/download/devspace-linux-arm64" && sudo install -c -m 0755 devspace /usr/local/bin
```
3. Install SOPS:
```bash
ORG="mozilla"
REPO="sops"
latest_release=$(curl -Ls "https://api.github.com/repos/${ORG}/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')

# AMD64
curl -L https://github.com/mozilla/sops/releases/download/v${latest_release}/sops_${latest_release}_amd64.deb -o sops.deb && sudo apt-get install ./sops.deb && rm sops.deb

# ARM64
curl -L https://github.com/mozilla/sops/releases/download/v${latest_release}/sops_${latest_release}_arm64.deb -o sops.deb && sudo apt-get install ./sops.deb && rm sops.deb
```

4. Generate a GPG key:
```bash
gpg --gen-key
#answer the questions
```

5. Create a SOPS configuration file:
```bash
first_pgp_key=$(gpg --list-secret-keys --keyid-format LONG | grep -m1 '^sec' | awk '{print $2}' | cut -d '/' -f2)

cat <<EOF > .sops.yaml
creation_rules:
- encrypted_regex: "^(data|stringData)$"
  pgp: >-
    ${first_pgp_key}
EOF
```

6. Create an encrypted Kubernetes ConfigMap with your OpenAI API key:

First, create an OPENAI_API_KEY environment variable and configure it with your OpenAI API key.
```bash
devspace run encrypt-openai-secret
```

4. Specify a namespace to use with DevSpace
```bash
devspace use namespace dev
```
5. Run devspace in dev mode
```bash
devspace dev
```

6. Generate a lot of jokes:

First, expose the `joke-server` pod or service using port forwarding.

```bash
kubectl -n dev port-forward svc/joke-server 8080:80
```
Then, execute the following command:

```bash 
for i in {1..10}; do curl http://localhost:8080/joke; echo -e; done
```

7. Modify the code

Your local repository is synchronized with the project files within the joke-worker pod. Modify the file `internal/joke/joke.go` and change the code so the joke generated is now a Chuck Norris joke. Replace the line:
```go
Prompt: "Tell me a dad joke"
```
With the line:
```go
Prompt: "Tell me a Chuck Norris joke"
```
Then save the file. The joke-worker pod will automatically recompile and restart the binary. Now, when you run the curl command again, you should see a Chuck Norris joke instead of a dad joke (provided you have generated less than 20 jokes, as the program will retrieve jokes from the Redis cache after that, for a time defined in `constants.RedisTTL`):
  
  ```bash
  for i in {1..10}; do curl http://localhost:8080/joke; echo -e; done
  ```
## License 📄
This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.
