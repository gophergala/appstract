# AppStract

It can be an overwhelming experience to join an open source project that has lots of code in it. You might start by reading documentation or even dive into the code directly. But sometimes you just want a quick overview. ''AppStract'' provides the solution.

''AppStract'' analyses go-code from a github repository. This analyses results in a graph that visualizes the structure of the go program. In the graph two functions are connected if one function calls another, thus creating an abstract for an entire go program.