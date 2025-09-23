package arboreal

import (
	"bytes"
	"testing"
)

func TestTextBlock_Value(t *testing.T) {
	var txt textBlock

	txt.Text = "This is a test"

	if txt.Value(AnnotatedMessages{}) != txt.Text {
		t.Fail()
	}
}

func TestAnnotationBlock_Value(t *testing.T) {
	var n annotationBlock

	n.Name = "Test"

	if n.Value(AnnotatedMessages{
		{
			Annotations: map[string]Annotation{
				"Test": Annotation{
					Name: "Test",
					Data: "Super Awesome!",
				},
			},
		},
	}) != "Super Awesome!" {
		t.Fail()
	}
}

func TestMultiAnnotationBlock(t *testing.T) {
	var m multiAnnotationBlock

	m.Parse("User preference: preference?")

	value := m.Value(AnnotatedMessages{
		{
			Annotations: map[string]Annotation{
				"preference": Annotation{
					Name: "preference",
					Data: "always address me as poopy butt",
				},
			},
		},
	})

	if value != "User preference: always address me as poopy butt" {
		t.Errorf("Unexpected value: %s", value)
	}

	emptyValue := m.Value(AnnotatedMessages{})
	if emptyValue != "" {
		t.Errorf("Unexpected value: %s", emptyValue)
	}
}

func TestDoubleQuestionMark(t *testing.T) {
	var m multiAnnotationBlock

	m.Parse("This is a test??")

	value := m.Value(AnnotatedMessages{})
	if value != "This is a test?" {
		t.Errorf("Unexpected value: %s", value)
	}
}

func TestAnnotationTemplate(t *testing.T) {
	var template AnnotationTemplate

	template.Parse("This is a test. Here is a basic annotation: {{ basic }}\n{{ This is a conditional? }}")

	var buf bytes.Buffer
	template.Execute(&buf, AnnotatedMessages{
		{
			Annotations: map[string]Annotation{
				"basic": Annotation{
					Name: "basic",
					Data: "hello world",
				},
				"conditional": Annotation{
					Name: "conditional",
					Data: "good idea",
				},
			},
		},
	})

	if buf.String() != "This is a test. Here is a basic annotation: hello world\nThis is a good idea" {
		t.Errorf("Unexpected value: %s", buf.String())
	}
}
