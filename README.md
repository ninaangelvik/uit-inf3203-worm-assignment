# Wormgate
An example wormgate and segment code for the [first
assignment](https://github.com/uit-inf-3200/Project-1) of the INF-3200
Distributed Systems Fundamentals course fall 2014. The Wormgate
([wormgate.go](https://github.com/uit-inf-3200/Wormgate/blob/master/wormgate.go))
is a simple program that starts a HTTP server and waits for a POST request to
`/segment` or `/wormgate`. Local worm segments use the `/segment` to spread to
new tiles, and remote wormgates use `/wormgate` to order wormgates to start a
segment. 

The `/segment` handler receives any source code from a segment and forwards it
to all the tiles on the display wall. 

The wormgate stores whatever it receives at `/wormgate` to
```/tmp/wormgate/[random hex string]/tmp.tar.gz```, attempts to extract this and
runs ```hello-world-graphic```. 

Please note that the wormgate starts up on port *8181*, your wormgate should
choose a different one! 

# Feedback
Please don't hesitate with asking questions or giving feedback in the [Issues
section](https://github.com/uit-inf-3200/Wormgate/issues)! If you feel that the
code could be better, submit a [pull
request](https://help.github.com/articles/using-pull-requests)! 

# How To

- Start the wormgates from `rocksvv`

```
    ansible tiles -m shell -B 60 -a 'export DISPLAY=:0 && go run /WORMGATEDIR/wormgate.go'
``` 
where `WORMGATEDIR` is where you placed the wormgate. 

- Send the segment code from a tile

```
go run segment.go
```

The source code for the segment code can be found in
[segment.go](https://github.com/uit-inf-3200/Wormgate/blob/master/segment.go),
and the code that is shipped to the tiles is found in 
[hello-world-graphic.go](https://github.com/uit-inf-3200/Wormgate/blob/master/hello-world-graphic.go),

NB that you might have to rebuild the `hello-world-graphic` binary (do this if
you get a weird exec error from the wormgate):

```
go build hello-world-graphic.go
```

# Prerequisites
The segment code uses [go-qml](https://github.com/go-qml/qml) to draw on the
screen. See their [readme](https://github.com/go-qml/qml/blob/v1/README.md) for
installation instructions. Everything should be installed for you on the display
wall. 
