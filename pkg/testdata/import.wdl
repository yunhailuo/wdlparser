version 1.1

import "test.wdl"
import "http://example.com/lib/analysis_tasks" as analysis
import "https://example.com/lib/stdlib.wdl"
  alias Parent as Parent2
  alias Child as Child2
  alias GrandChild as GrandChild2

workflow HelloWorld {
    call WriteGreeting
}

task WriteGreeting {
    command {
       echo "Hello world!"
    }
}