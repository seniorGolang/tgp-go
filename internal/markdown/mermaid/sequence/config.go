// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package sequence

type config struct {
	// mirrorActors turns on/off the rendering of actors
	// below the diagram as well as above it.
	// Default is false.
	mirrorActors bool
	// bottomMariginAdjustment Adjusts how far down the graph ended.
	// Wide borders styles with css could generate unwanted clipping which is why this config param exists.
	// Default is 1.
	bottomMariginAdjustment uint
	// actorFontSize is the font size of the actors.
	// Default is 14.
	actorFontSize uint
	// actorFontFamily sets the font family for the actor's description.
	// Default is "Open Sans", sans-serif.
	actorFontFamily string
	// actorFontWeight sets the font weight for the actor's description
	// Default is "Open Sans", sans-serif.
	actorFontWeight string
	// noteFontSize is the font size of the notes.
	// Default is 14.
	noteFontSize uint
	// noteFontFamily sets the font family for the note's description.
	// Default is "trebuchet ms", verdana, arial
	noteFontFamily string
	// noteFontWeight sets the font weight for the note's description.
	// Default is "trebuchet ms", verdana, arial
	noteFontWeight string
	// noteAlign sets the alignment of the note's description.
	// Default is "center"
	noteAlign string
	// messageFontSize is the font size of the messages.
	// Default is 16.
	messageFontSize uint
	// messageFontFamily sets the font family for actor<->actor messages
	// Default is "trebuchet ms", verdana, arial
	messageFontFamily string
	// messageFontWeight sets the font weight for actor<->actor messages
	// Default is "trebuchet ms", verdana, arial
	messageFontWeight string
}

func newConfig() *config {
	return &config{
		mirrorActors:            false,
		bottomMariginAdjustment: 1,
		actorFontSize:           14, //nolint:mnd
		actorFontFamily:         "Open Sans, sans-serif",
		actorFontWeight:         "Open Sans, sans-serif",
		noteFontSize:            14, //nolint:mnd
		noteFontFamily:          "trebuchet ms, verdana, arial",
		noteFontWeight:          "trebuchet ms, verdana, arial",
		noteAlign:               "center",
		messageFontSize:         16, //nolint:mnd
		messageFontFamily:       "trebuchet ms, verdana, arial",
		messageFontWeight:       "trebuchet ms, verdana, arial",
	}
}

type Option func(*config)

func WithMirrorActors(mirrorActors bool) Option {
	return func(c *config) {
		c.mirrorActors = mirrorActors
	}
}

func WithBottomMariginAdjustment(bottomMariginAdjustment uint) Option {
	return func(c *config) {
		c.bottomMariginAdjustment = bottomMariginAdjustment
	}
}

func WithActorFontSize(actorFontSize uint) Option {
	return func(c *config) {
		c.actorFontSize = actorFontSize
	}
}

func WithActorFontFamily(actorFontFamily string) Option {
	return func(c *config) {
		c.actorFontFamily = actorFontFamily
	}
}

func WithActorFontWeight(actorFontWeight string) Option {
	return func(c *config) {
		c.actorFontWeight = actorFontWeight
	}
}

func WithNoteFontSize(noteFontSize uint) Option {
	return func(c *config) {
		c.noteFontSize = noteFontSize
	}
}

func WithNoteFontFamily(noteFontFamily string) Option {
	return func(c *config) {
		c.noteFontFamily = noteFontFamily
	}
}

func WithNoteFontWeight(noteFontWeight string) Option {
	return func(c *config) {
		c.noteFontWeight = noteFontWeight
	}
}

func WithNoteAlign(noteAlign string) Option {
	return func(c *config) {
		c.noteAlign = noteAlign
	}
}

func WithMessageFontSize(messageFontSize uint) Option {
	return func(c *config) {
		c.messageFontSize = messageFontSize
	}
}

func WithMessageFontFamily(messageFontFamily string) Option {
	return func(c *config) {
		c.messageFontFamily = messageFontFamily
	}
}

func WithMessageFontWeight(messageFontWeight string) Option {
	return func(c *config) {
		c.messageFontWeight = messageFontWeight
	}
}
