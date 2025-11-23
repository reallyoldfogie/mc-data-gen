package loader

import "testing"

func TestMakePropsKey(t *testing.T) {
    cases := []struct {
        name  string
        in    map[string]string
        want  string
    }{
        {name: "empty", in: nil, want: ""},
        {name: "single", in: map[string]string{"facing": "north"}, want: "facing=north"},
        {name: "sorted", in: map[string]string{"b": "2", "a": "1"}, want: "a=1,b=2"},
    }

    for _, tc := range cases {
        got := MakePropsKey(tc.in)
        if got != tc.want {
            t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
        }
    }
}

