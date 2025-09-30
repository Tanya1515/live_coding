package main

// Надо держать в голове, что если к методу Do будет 
// конкурентный доступ, то будет data race 

// Также необходимо добавить корректное описание сигнатуры для метода Send(tr Trace)

var c int

type Trace struct {

}

type Sender interface {
    Send(tr Trace)
}

func Do(sender Sender, tr Trace) {
    c++
    if c == 100 {
        sender.Send(tr)
        c = 0        
    }
}