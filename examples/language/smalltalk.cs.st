Class: CookieMonster
Superclass: Monster
Category: Sesame Street
Instance variables: state hunger
 
private

    askForCookie
       ^ FillInTheBlank request: 'Give me cookie !!! (please)'
        
    complainAbout: anItem
    Transcript show: 'No want ', anItem printString.
    Transcript cr.
    self colour: #red
        
    isCookie: anItem
    " | serves as the OR operator"
    ^ ((anItem = 'cookie') | (anItem = #cookie))

actions

    eat: aCookie
    super eat: aCookie.
    self colour: #green
        
    nag
       | item |
          [self isAwake]
          whileTrue:
            [item := self askForCookie.
            (self isCookie: item)
                  ifTrue: [self eat: item]
                  ifFalse: [self complainAbout: item].
            (self isFull) ifTrue: [self sleep]]
        
    sleep
       self state: #asleep.
    self hunger: 0
        
    wakeUp
       self tummy: Bag new.
    self state: #awake.
    self hunger: (Random new next * 13).
    "Cookie Monsters are superstitious and never eat more than 
    13 cookies in one go !"
    self nag

queries

    isAsleep
       ^ state = #asleep
        
    isAwake
       ^ self isAsleep not
        
    isFull
       self isEmpty 
       ifFalse: [^ self tummy size >= self hunger] 
       ifTrue: [^false]

access

     
    hunger
       ^ hunger
        
    hunger: anIntegerNumberOfCookies
    hunger := anIntegerNumberOfCookies
        
    state
       ^ state
        
    state: aSymbol
    state := aSymbol

initialization

    initialize
       self state: #asleep.
    self hunger: nil.
    super initialize