// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package sequence

func (d *Diagram) LoopStart(description string) (out *Diagram) {
	d.body = append(d.body, "    loop "+description)
	return d
}

func (d *Diagram) LoopEnd() (out *Diagram) {
	d.body = append(d.body, "    end")
	return d
}

func (d *Diagram) AltStart(description string) (out *Diagram) {
	d.body = append(d.body, "    alt "+description)
	return d
}

func (d *Diagram) AltElse(description string) (out *Diagram) {
	d.body = append(d.body, "    else "+description)
	return d
}

func (d *Diagram) AltEnd() (out *Diagram) {
	d.body = append(d.body, "    end")
	return d
}

func (d *Diagram) OptStart(description string) (out *Diagram) {
	d.body = append(d.body, "    opt "+description)
	return d
}

func (d *Diagram) OptEnd() (out *Diagram) {
	d.body = append(d.body, "    end")
	return d
}

func (d *Diagram) ParallelStart(description string) (out *Diagram) {
	d.body = append(d.body, "    par "+description)
	return d
}

func (d *Diagram) ParallelAnd(description string) (out *Diagram) {
	d.body = append(d.body, "    and "+description)
	return d
}

func (d *Diagram) ParallelEnd() (out *Diagram) {
	d.body = append(d.body, "    end")
	return d
}

func (d *Diagram) CriticalStart(description string) (out *Diagram) {
	d.body = append(d.body, "    critical "+description)
	return d
}

func (d *Diagram) CriticalOption(description string) (out *Diagram) {
	d.body = append(d.body, "    option "+description)
	return d
}

func (d *Diagram) CriticalEnd() (out *Diagram) {
	d.body = append(d.body, "    end")
	return d
}

func (d *Diagram) BreakStart(description string) (out *Diagram) {
	d.body = append(d.body, "    break "+description)
	return d
}

func (d *Diagram) BreakEnd() (out *Diagram) {
	d.body = append(d.body, "    end")
	return d
}
