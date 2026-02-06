// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package sequence

import (
	"fmt"
)

func (d *Diagram) LoopStart(description string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    loop %s", description))
	return d
}

func (d *Diagram) LoopEnd() *Diagram {
	d.body = append(d.body, "    end")
	return d
}

func (d *Diagram) AltStart(description string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    alt %s", description))
	return d
}

func (d *Diagram) AltElse(description string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    else %s", description))
	return d
}

func (d *Diagram) AltEnd() *Diagram {
	d.body = append(d.body, "    end")
	return d
}

func (d *Diagram) OptStart(description string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    opt %s", description))
	return d
}

func (d *Diagram) OptEnd() *Diagram {
	d.body = append(d.body, "    end")
	return d
}

func (d *Diagram) ParallelStart(description string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    par %s", description))
	return d
}

func (d *Diagram) ParallelAnd(description string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    and %s", description))
	return d
}

func (d *Diagram) ParallelEnd() *Diagram {
	d.body = append(d.body, "    end")
	return d
}

func (d *Diagram) CriticalStart(description string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    critical %s", description))
	return d
}

func (d *Diagram) CriticalOption(description string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    option %s", description))
	return d
}

func (d *Diagram) CriticalEnd() *Diagram {
	d.body = append(d.body, "    end")
	return d
}

func (d *Diagram) BreakStart(description string) *Diagram {
	d.body = append(d.body, fmt.Sprintf("    break %s", description))
	return d
}

func (d *Diagram) BreakEnd() *Diagram {
	d.body = append(d.body, "    end")
	return d
}
