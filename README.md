# Wormgate
An example wormgate and segment code for the first assignment of the INF-3200
Distributed Systems Fundamentals course fall 2014. The Wormgate ([wormgate.go](https://github.com/uit-inf-3200/Wormgate/blob/master/wormgate.go)) is
a simple program that starts a HTTP server and waits for a POST request to
```/segment```. It stores whatever it receives to ```/tmp/wormgate/[random hex string]/tmp.tar.gz```,
attempts to extract this and runs ```hello-world-graphic```. 

# Feedback
Please don't hesitate with asking questions or giving feedback in the [Issues section](https://github.com/uit-inf-3200/Wormgate/issues)! If you feel that the code could be better, submit a [pull request](https://help.github.com/articles/using-pull-requests)! 

# How To

- Start the wormgate

``` 
go run wormgate.go
``` 

- Send the segment code

```
go run ctrlman.go
```

The source code for the segment code can be found in [hello-world-graphic.go](https://github.com/uit-inf-3200/Wormgate/blob/master/hello-world-graphic.go)

NB that you might have to rebuild the segment binary (do this if you get a weird exec error from the wormgate):

```
go build hello-world-graphic.go
```

# Prerequisites
The segment code uses [go-qml](https://github.com/go-qml/qml) to draw on the screen. See their [readme](https://github.com/go-qml/qml/blob/v1/README.md) for installation instructions. Everything should be installed for you on the display wall. 
