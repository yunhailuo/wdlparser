version 1.1

workflow HelloWorld {
    call WriteGreeting
}

task WriteGreeting {
    command <<<
        echo "Hello world!"
    >>>
}