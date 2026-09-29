package contract

type interfaceM interface{ M() }

type T struct{}

func (*T) M() {}

func example() (valueImplements, pointerImplements, embeddedValueImplements bool) {
	_, valueImplements = any(T{}).(interfaceM)
	_, pointerImplements = any(&T{}).(interfaceM)
	embedded := struct{ *T }{}
	_, embeddedValueImplements = any(embedded).(interfaceM)
	return
}
