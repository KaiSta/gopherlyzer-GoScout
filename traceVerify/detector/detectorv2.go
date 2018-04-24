package race

import (
	"fmt"

	"../parser"
	"../util"
	"./eraser"
	"./fastTrack"
	"./raceTrack"
	"./report"
	"./traceReplay"
	"./twinTrack"
)

func startThreads(m *traceReplay.Machine) {
	for {
		threads := m.GetThreadStarts()
		if len(threads) == 0 {
			return
		}
		for k, v := range threads {
			s := &traceReplay.SyncPair{T1: k, T2: v, IsGoStart: true}
			for _, l := range traceReplay.EvListener {
				l.Put(m, s)
			}
		}
	}
}

func replay(m *traceReplay.Machine, jsonFlag, plain, bench bool) {
	for {
		//m.StartAllthreads()
		startThreads(m)

		pairs := m.GetNextActionWCommLink()

		if len(pairs) == 0 {
			break
		}

		//pairs := m.GetNextRandomActionWCommLink()

		//for i := range pairs {
		m.UpdateChanVc(&pairs[0])
		//}
		for _, l := range traceReplay.EvListener {
			l.Put(m, &pairs[0])
		}
	}
}

func RunFastTrack(tracePath string, json, plain, bench bool) {
	items := parser.ParseTrace(tracePath)
	threads := createThreads(items)

	// for _, t := range threads {
	// 	fmt.Println(t.ID)
	// 	for _, e := range t.Events {
	// 		fmt.Printf("\t%v\n", e)
	// 	}
	// }

	asyncChans := getAsyncChans2(items)
	closedChans := make(map[string]struct{})
	closedChans["0"] = struct{}{}
	m := &traceReplay.Machine{threads, nil,
		closedChans, make(map[string]util.VectorClock),
		false,
		nil, nil,
		make(map[string]*util.VarState3),
		asyncChans,
		make(map[string]*util.ChanState),
		make([]traceReplay.SelectStore, 0)}
	traceReplay.EvListener = []traceReplay.EventListener{
		&fastTrack.ListenerSelect{},
		&fastTrack.ListenerSync{},
		&fastTrack.ListenerAsyncSnd{},
		&fastTrack.ListenerAsyncRcv{},
		&fastTrack.ListenerChanClose{},
		&fastTrack.ListenerOpClosedChan{},
		&fastTrack.ListenerDataAccess2{},
		&fastTrack.ListenerGoStart{},
	}
	replay(m, json, plain, bench)
	m.PostProcessing()
	report.ReportAlts()
	fmt.Printf("\n\n")
	//report.Events()
	report.AlternativesFromReport()
}

func RunEraser(tracePath string, json, plain, bench bool) {
	items := parser.ParseTrace(tracePath)
	threads := createThreads(items)

	asyncChans := getAsyncChans2(items)
	closedChans := make(map[string]struct{})
	closedChans["0"] = struct{}{}
	m := &traceReplay.Machine{threads, nil,
		closedChans, make(map[string]util.VectorClock),
		false,
		nil, nil,
		make(map[string]*util.VarState3),
		asyncChans,
		make(map[string]*util.ChanState),
		make([]traceReplay.SelectStore, 0)}
	traceReplay.EvListener = []traceReplay.EventListener{
		&eraser.ListenerSelect{},
		&eraser.ListenerSync{},
		&eraser.ListenerAsyncSnd{},
		&eraser.ListenerAsyncRcv{},
		&eraser.ListenerChanClose{},
		&eraser.ListenerOpClosedChan{},
		&eraser.ListenerDataAccess{},
		&eraser.ListenerGoStart{},
	}
	replay(m, json, plain, bench)
}

func RunRaceTrack(tracePath string, json, plain, bench bool) {
	items := parser.ParseTrace(tracePath)
	threads := createThreads(items)

	asyncChans := getAsyncChans2(items)
	closedChans := make(map[string]struct{})
	closedChans["0"] = struct{}{}
	m := &traceReplay.Machine{threads, nil,
		closedChans, make(map[string]util.VectorClock),
		false,
		nil, nil,
		make(map[string]*util.VarState3),
		asyncChans,
		make(map[string]*util.ChanState),
		make([]traceReplay.SelectStore, 0)}
	traceReplay.EvListener = []traceReplay.EventListener{
		&raceTrack.ListenerSelect{},
		&raceTrack.ListenerSync{},
		&raceTrack.ListenerAsyncSnd{},
		&raceTrack.ListenerAsyncRcv{},
		&raceTrack.ListenerChanClose{},
		&raceTrack.ListenerOpClosedChan{},
		&raceTrack.ListenerDataAccess{},
		&raceTrack.ListenerGoStart{},
	}
	replay(m, json, plain, bench)
}

func RunTwinTrack(tracePath string, json, plain, bench bool) {
	items := parser.ParseTrace(tracePath)
	threads := createThreads(items)

	asyncChans := getAsyncChans2(items)
	closedChans := make(map[string]struct{})
	closedChans["0"] = struct{}{}
	m := &traceReplay.Machine{threads, nil,
		closedChans, make(map[string]util.VectorClock),
		false,
		nil, nil,
		make(map[string]*util.VarState3),
		asyncChans,
		make(map[string]*util.ChanState),
		make([]traceReplay.SelectStore, 0)}
	traceReplay.EvListener = []traceReplay.EventListener{
		&twinTrack.ListenerSelect{},
		&twinTrack.ListenerSync{},
		&twinTrack.ListenerAsyncSnd{},
		&twinTrack.ListenerAsyncRcv{},
		&twinTrack.ListenerChanClose{},
		&twinTrack.ListenerOpClosedChan{},
		&twinTrack.ListenerDataAccess{},
		&twinTrack.ListenerGoStart{},
	}
	replay(m, json, plain, bench)
}
