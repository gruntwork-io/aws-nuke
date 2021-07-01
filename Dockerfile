# Check the README.md for build and run instructions

####### Builder ########
FROM golang:1.15-alpine as builder
#Preparing working directory and copying files
WORKDIR /wrk
COPY . . 

#You can overrride by setting the CIRCLE_TAG environment variable. 0.0.0 is just a default, so you'll notice if you forget
ARG VERSION=0.0.0
RUN go build -o dest/cloud-nuke -ldflags="-X main.VERSION=$VERSION"

####### Runner ########
#Choosing alpine base as runner, since we don't need the go compiler to run the program
FROM alpine:3 as runner

#Set working directory
WORKDIR /wrk
#Create non-root user, to make the container safer to run
RUN addgroup -S app && adduser -S -G app app
#Own the working folder
RUN chown app:app . 
#Switch to the new non-root user
USER app
#Copy over just the binary from the builder image and change the owner to the non-root user
COPY --from=builder --chown=app:app /wrk/dest/ .
#Make the binary executable
RUN chmod +x cloud-nuke
#Set the entrypoint so that cloud-nuke is the default action
ENTRYPOINT ["./cloud-nuke"]
