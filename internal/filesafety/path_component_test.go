package filesafety

import "testing"

func TestIsSafePathComponent(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "image", value: "image.png", want: true},
		{name: "generated image", value: "1_user_mainQuestion_42_image.jpg", want: true},
		{name: "punctuation", value: "a-b_c.d", want: true},
		{name: "empty", value: "", want: false},
		{name: "dot", value: ".", want: false},
		{name: "dot dot", value: "..", want: false},
		{name: "traversal", value: "../image.png", want: false},
		{name: "forward slash", value: "subdir/image.png", want: false},
		{name: "backslash", value: `subdir\image.png`, want: false},
		{name: "absolute", value: "/absolute.png", want: false},
		{name: "windows absolute", value: `C:\absolute.png`, want: false},
		{name: "nul", value: "image\x00.png", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsSafePathComponent(test.value); got != test.want {
				t.Fatalf("IsSafePathComponent(%q)=%t, want %t", test.value, got, test.want)
			}
		})
	}
}
