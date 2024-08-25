## how to build image
1. run `make build` to build the image
2. run `make push` to push the image to the registry

## how to deploy the backend server
1. copy `run.sh` file to the server
1. edit the `run.sh` file to set the correct environment variables:
   1. `AUTH_TOKEN` - the token that the client will use to authenticate with openai platform
   2. `ASSISTANT_ID` - the id of the assistant that the backend will use to interact with the openai platform
1. run `chmod +x run.sh` to make the file executable
1. run `./run.sh` to start the server

## how to stop the backend server
1. copy `stop.sh` file to the server
2. run `chmod +x stop.sh` to make the file executable
3. run `./stop.sh` to stop the server
4. if you want to change the environment variables, you can edit the `run.sh` file and restart the server