version 1.1

workflow HelloWorld {
    call Greeting as hello {
        input:
            first_name = first_name,
            last_name = "Luo"
    }
    call Goodbye after hello { input: first_name = "Yunhai", }
}