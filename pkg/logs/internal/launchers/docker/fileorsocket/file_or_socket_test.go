// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package fileorsocket

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type testcase struct {
	name    string
	decider Decider
	result  LogFrom
}

var cases []testcase = []testcase{
	testcase{
		name:    "not TailFromFile",
		decider: Decider{TailFromFile: false},
		result:  Socket,
	},
	testcase{
		name:    "ForceTailFromFile",
		decider: Decider{TailFromFile: true, ForceTailFromFile: true, SocketInRegistry: true},
		result:  File,
	},
	testcase{
		name:    "SocketInRegistry",
		decider: Decider{TailFromFile: true, ForceTailFromFile: false, SocketInRegistry: true},
		result:  Socket,
	},
}

func Test(t *testing.T) {
	for _, testcase := range cases {
		t.Run(testcase.name, func(t *testing.T) {
			require.Equal(t, testcase.result, testcase.decider.Decide(), testcase.name)
		})
	}
}
