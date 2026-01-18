; An example comes from https://www.re-bol.com/rebol.html#section-11
; This is a slightly edited version of the 3D Maze program (raycasting engine) by Olivier Auverlot
REBOL [title: "3D Maze - Ray Casting Example"] 

px: 9 * 1024  py: 11 * 1024 stride: 2 heading: 0 turn: 5
laby: [
    [ 8 7 8 7 8 7 8 7 8 7 8 7 ]
    [ 7 0 0 0 0 0 0 0 13 0 0 8 ]
    [ 8 0 0 0 12 0 0 0 14 0 9 7 ]
    [ 7 0 0 0 12 0 4 0 13 0 0 8 ]
    [ 8 0 4 11 11 0 3 0 0 0 0 7 ]
    [ 7 0 3 0 12 3 4 3 4 3 0 8 ]
    [ 8 0 4 0 0 0 3 0 3 0 0 7 ]
    [ 7 0 3 0 0 0 4 0 4 0 9 8 ]
    [ 8 0 4 0 0 0 0 0 0 0 0 7 ]
    [ 7 0 5 6 5 6 0 0 0 0 0 8 ]
    [ 8 0 0 0 0 0 0 0 0 0 0 7 ]
    [ 8 7 8 7 8 7 8 7 8 7 8 7 ]
]
ctable: []
for a 0 (718 + 180) 1 [
    append ctable to-integer (((cosine a ) * 1024) / 20)
]
palette: [
    0.0.128 0.128.0 0.128.128
    0.0.128 128.0.128 128.128.0 192.192.192
    128.128.128 0.0.255 0.255.0 255.255.0
    0.0.255 255.0.255 0.255.255 255.255.255
]
get-angle: func [ v ] [ pick ctable (v + 1) ]
retrace: does [
    clear display/effect/draw
    xy1: xy2: 0x0
    angle: remainder (heading - 66) 720
    if angle < 0 [ angle: angle + 720 ]
    for a angle (angle + 89) 1 [
        xx: px
        yy: py
        stepx: get-angle a + 90
        stepy: get-angle a
        l: 0
        until [
            xx: xx - stepx
            yy: yy - stepy
            l: l + 1
            column: make integer! (xx / 1024)
            line: make integer! (yy / 1024)
            laby/:line/:column <> 0
        ]
        h: make integer! (1800 / l)
        xy1/y: 200 - h
        xy2/y: 200 + h
        xy2/x: xy1/x + 6
        color: pick palette laby/:line/:column
        append display/effect/draw reduce [
            'pen color
            'fill-pen color
            'box xy1 xy2
        ]
        xy1/x: xy2/x + 2  ; set to 1 for smooth walls
    ]
]
player-move: function [ /backwards ] [ mul ] [
    either backwards [ mul: -1 ] [ mul: 1 ]
    newpx: px - ((get-angle (heading + 90)) * stride * mul)
    newpy: py - ((get-angle heading) * stride * mul)
    c: make integer! (newpx / 1024)
    l: make integer! (newpy / 1024)
    if laby/:l/:c = 0 [
        px: newpx
        py: newpy
        refresh-display
    ]
]
evt-key: function [ f event ] [] [
    if (event/type = 'key) [
        switch event/key [
            up [ player-move ]
            down [ player-move/backwards ]
            left [
                heading: remainder (heading + (720 - turn)) 720
                refresh-display
            ]
            right [
                heading: remainder (heading + turn) 720
                refresh-display
            ]
        ]
    ]
    event
]
insert-event-func :evt-key
refresh-display: does [
    retrace
    show display
]
screen: layout [
    display: box 720x400 effect [
        gradient 0x1 0.0.0 128.128.128
        draw []
    ]
    edge [
        size: 1x1
        color: 255.255.255
    ]
]
refresh-display
view screen
