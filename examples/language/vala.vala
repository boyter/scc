// test for vala

class Demo.Demo : GLib.Object {
    public static int main(string[] args) {
        int[] arr = {1, 2, 3, 4, 5, 6, 7, 8, 9};
        foreach (var i in arr) {
            foreach (var j in arr[0:i]) {
                var n = i * j;
                stdout.printf(@"$j * $i = $n ");
            }
            stdout.printf("\n");
        }
        stdout.printf("""
        Test for "verbatim strings".
        \r \n \a \b
        """);
        stdout.printf("\n");
        return 0;
    }
}
