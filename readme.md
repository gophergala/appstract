# AppStract

It can be an overwhelming experience to join an open source project that has a lot of code in it. You might start by reading documentation or even dive into the code directly. But sometimes you just want a quick overview. <i>AppStract</i> provides the solution.

<i>AppStract</i> analyzes go code from a github repository. This analysis results in a graph that visualizes the structure of the go program. In the graph two functions are connected if one function calls another, thus creating an abstract for the entire go program.

Check it out at [go-appstract.appspot.com](http://go-appstract.appspot.com).
