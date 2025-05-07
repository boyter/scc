class Person {
    constructor(name) {
        this.name = name;
    }
}

class Student extends Person {
    constructor(name, id) {
        super(name);
        this.id = id;
    }
}

function PrintName(student) {
    console.log(student.name);
}

function PrintID(student) {
    console.log(student.id);
}

const bob = new Student("Robert", 12345);
PrintName(bob)  // Robert
PrintID(bob)    // 12345
