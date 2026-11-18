package cloud

import (
	"slices"
	"time"

	pb "github.com/amir20/dozzle/proto/cloud"
)

// VisualTypes are the pictures the chat pane can draw. They are sent with every
// turn (ChatTurn.visual_types): cloud offers the assistant its picture tools
// only for these, so adding a type here is how the pane opts into it, and a
// type cloud adds later is never sent to a build that cannot draw it.
var VisualTypes = []string{"memory_vs_limit", "cpu_over_time", "pattern_count", "container_ranking", "logs", "container_events"}

// ChatVisual is pb.ChatVisual for the browser. Every number was measured by
// cloud; the pane draws it as it came. See cloud.proto for the fields.
type ChatVisual struct {
	Type      string `json:"type"`
	Container string `json:"container,omitempty"`
	HostID    string `json:"hostId,omitempty"`
	HostName  string `json:"hostName,omitempty"`
	Query     string `json:"query,omitempty"`
	Metric    string `json:"metric,omitempty"`
	// Milliseconds, like MetricPoint.TS, so the browser reads them into a Date.
	From int64 `json:"from"`
	To   int64 `json:"to"`

	Facts       *ChatVisualFacts     `json:"facts,omitempty"`
	Total       *int64               `json:"total,omitempty"`
	Rows        []ChatVisualRankRow  `json:"rows,omitempty"`
	Lines       []ViewLogLine        `json:"lines,omitempty"`
	Volume      []SearchVolumeBucket `json:"volume,omitempty"`
	StepNs      int64                `json:"stepNs,omitempty"`
	Points      []MetricPoint        `json:"points,omitempty"`
	MemoryLimit int64                `json:"memoryLimit,omitempty"`
}

type ChatVisualFacts struct {
	Exits          int32                `json:"exits"`
	ExitCodes      []ChatVisualExitCode `json:"exitCodes,omitempty"`
	OOMs           int32                `json:"ooms"`
	Restarts       int32                `json:"restarts"`
	EventsRecorded bool                 `json:"eventsRecorded"`
	Capped         bool                 `json:"capped"`
	MemoryPeak     int64                `json:"memoryPeak,omitempty"`
}

type ChatVisualExitCode struct {
	Code  string `json:"code"`
	Count int32  `json:"count"`
}

type ChatVisualRankRow struct {
	Container string  `json:"container"`
	HostID    string  `json:"hostId,omitempty"`
	HostName  string  `json:"hostName,omitempty"`
	Value     float64 `json:"value"`
}

// visualsFromProto converts the pictures a turn ended with. A type this build
// does not draw is dropped rather than handed to a pane that has no component
// for it; cloud should never send one, since the turn listed what it can draw.
func visualsFromProto(in []*pb.ChatVisual) []ChatVisual {
	if len(in) == 0 {
		return nil
	}
	out := make([]ChatVisual, 0, len(in))
	for _, v := range in {
		if !slices.Contains(VisualTypes, v.GetType()) {
			continue
		}
		// A CPU picture is its series and nothing else. Cloud sends it without
		// one when the read behind it failed or came back empty, and drawn, it
		// would be a container's name over an empty box.
		if v.GetType() == "cpu_over_time" && len(v.GetPoints()) == 0 {
			continue
		}
		cv := ChatVisual{
			Type:        v.GetType(),
			Container:   v.GetContainer(),
			HostID:      v.GetHostId(),
			HostName:    v.GetHostName(),
			Query:       v.GetQuery(),
			Metric:      v.GetMetric(),
			From:        v.GetFromNs() / 1e6,
			To:          v.GetToNs() / 1e6,
			Total:       v.Total,
			StepNs:      v.GetStepNs(),
			MemoryLimit: v.GetMemoryLimit(),
		}
		if f := v.GetFacts(); f != nil {
			facts := &ChatVisualFacts{
				Exits:          f.GetExits(),
				OOMs:           f.GetOoms(),
				Restarts:       f.GetRestarts(),
				EventsRecorded: f.GetEventsRecorded(),
				Capped:         f.GetCapped(),
				MemoryPeak:     f.GetMemoryPeak(),
			}
			for _, c := range f.GetExitCodes() {
				facts.ExitCodes = append(facts.ExitCodes, ChatVisualExitCode{Code: c.GetCode(), Count: c.GetCount()})
			}
			cv.Facts = facts
		}
		for _, r := range v.GetRows() {
			cv.Rows = append(cv.Rows, ChatVisualRankRow{Container: r.GetContainer(), HostID: r.GetHostId(), HostName: r.GetHostName(), Value: r.GetValue()})
		}
		for _, l := range v.GetLines() {
			cv.Lines = append(cv.Lines, lineFromProto(l))
		}
		for _, b := range v.GetVolume() {
			cv.Volume = append(cv.Volume, SearchVolumeBucket{TimestampNs: b.GetTimestampNs(), Level: b.GetLevel(), Count: b.GetCount()})
		}
		if len(v.GetPoints()) > 0 {
			// The same conversion the metrics panel's points go through, so the
			// chart reads them exactly as it reads its own.
			cv.Points = metricResultFromProto(&pb.GetContainerMetricsResponse{Points: v.GetPoints()}).Points
		}
		out = append(out, cv)
	}
	return out
}

// lineFromProto is lineToProto's inverse.
func lineFromProto(l *pb.ViewLogLine) ViewLogLine {
	line := ViewLogLine{Level: l.GetLevel(), ContainerID: l.GetContainerId(), Message: l.GetMessage()}
	if ns := l.GetTimestampNs(); ns != 0 {
		line.Timestamp = time.Unix(0, ns).UTC().Format(time.RFC3339)
	}
	return line
}
