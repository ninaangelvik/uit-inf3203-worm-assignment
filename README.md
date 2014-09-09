# WARNING
[We had some last minutes changes to the
project](https://github.com/uit-inf-3200/Project-1/commit/8fae079d360aa6c098cdfab1b0871c149e62fd5a),
so the current implementation will not reflect on how we would like the
Wormgates to work. The previous implementation required each segment to contact
a remote wormgate to start up a segment, but we want to hide all of this into
the wormgates. Segments should contact their local wormgate which then does all
the work of starting up a segment on a (possibly) remote computer. We will
update the repository throughout the week to reflect the changes. If you'd like
you can fix it yourself and send a pull request to us! 

# Wormgate
An example wormgate and segment code for the [first assignment](https://github.com/uit-inf-3200/Project-1) of the INF-3200
Distributed Systems Fundamentals course fall 2014. The Wormgate ([wormgate.go](https://github.com/uit-inf-3200/Wormgate/blob/master/wormgate.go)) is
a simple program that starts a HTTP server and waits for a POST request to
```/segment```. It stores whatever it receives to ```/tmp/wormgate/[random hex string]/tmp.tar.gz```,
attempts to extract this and runs ```hello-world-graphic```. Please note that
the wormgate starts up on port *8181*, your wormgate should choose a different
one! 

# Feedback
Please don't hesitate with asking questions or giving feedback in the [Issues
section](https://github.com/uit-inf-3200/Wormgate/issues)! If you feel that the
code could be better, submit a [pull
request](https://help.github.com/articles/using-pull-requests)! 

# How To

- Start the wormgate

``` 
go run wormgate.go
``` 

- Send the segment code

```
go run ctrlman.go
```

The source code for the segment code can be found in
[hello-world-graphic.go](https://github.com/uit-inf-3200/Wormgate/blob/master/hello-world-graphic.go)

NB that you might have to rebuild the segment binary (do this if you get a weird
exec error from the wormgate):

```
go build hello-world-graphic.go
```

# Prerequisites
The segment code uses [go-qml](https://github.com/go-qml/qml) to draw on the
screen. See their [readme](https://github.com/go-qml/qml/blob/v1/README.md) for
installation instructions. Everything should be installed for you on the display
wall. 
