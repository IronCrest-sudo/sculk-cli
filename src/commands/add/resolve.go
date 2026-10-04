package add

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"sculk-cli/src/version"

	"github.com/go-git/go-git/v6"
	gitconfig "github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/storage/memory"

	"charm.land/log/v2"
)

// RefKind tells a branch from a tag.
type RefKind string

const (
	RefBranch RefKind = "branch"
	RefTag    RefKind = "tag"
)

// RemoteRef is one publishable ref of a library repository.
//
// sculk reads versions straight out of ref names. The publishing convention is
//
//	<version>              e.g. 1.0.0
//	<version>/<gameVer>    e.g. 1.0.0/26.2
//
// either as a branch or as a tag, with a leading "v" tolerated.
//
// Note the git limitation this has to work around: git stores refs as files,
// so a repository cannot hold both refs/heads/1.0.0 and
// refs/heads/1.0.0/26.2 - the second needs 1.0.0 to be a directory. sculk
// therefore also accepts these equivalent spellings, which are all
// single-level refs and cannot collide:
//
//	1.0.0+26.2
//	1.0.0_26.2
//
// A hyphen is deliberately NOT a separator, because game versions themselves
// contain one ("26.4-snapshot-1").
type RemoteRef struct {
	Name        string
	Kind        RefKind
	RefName     plumbing.ReferenceName
	Version     version.Version
	GameVersion version.Version
}

// refSeparators are the characters that may separate a version from a game
// version inside a ref name, in the order sculk tries them.
var refSeparators = []string{"/", "+", "_"}

// splitRefName cuts a ref name into its version and game-version halves on the
// first separator it finds.
func splitRefName(name string) (versionPart string, gamePart string) {
	cut := -1
	sepLen := 0
	for _, sep := range refSeparators {
		if i := strings.Index(name, sep); i >= 0 && (cut < 0 || i < cut) {
			cut, sepLen = i, len(sep)
		}
	}
	if cut < 0 {
		return name, ""
	}
	return name[:cut], name[cut+sepLen:]
}

// FullName is the canonical "<version>/<gameVer>" spelling of this ref.
func (r RemoteRef) FullName() string {
	if r.GameVersion.IsZero() {
		return r.Version.String()
	}
	return r.Version.String() + "/" + r.GameVersion.String()
}

// refCache avoids hitting the git host twice for the same repository during a
// single CLI invocation (`sculk add` resolves, then metadata is rebuilt).
var (
	refCacheMu sync.Mutex
	refCache   = map[string][]RemoteRef{}
)

// ClearRefCache drops the memoised remote ref listings. Exposed for tests.
func ClearRefCache() {
	refCacheMu.Lock()
	defer refCacheMu.Unlock()
	refCache = map[string][]RemoteRef{}
}

// ListRemoteRefs asks the git host for every branch and tag of repoURL.
func ListRemoteRefs(repoURL string) ([]RemoteRef, error) {
	refCacheMu.Lock()
	if cached, ok := refCache[repoURL]; ok {
		refCacheMu.Unlock()
		return cached, nil
	}
	refCacheMu.Unlock()

	remote := git.NewRemote(memory.NewStorage(), &gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})

	raw, err := remote.List(&git.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not list refs of %s: %w", repoURL, err)
	}

	refs := make([]RemoteRef, 0, len(raw))
	for _, r := range raw {
		name := r.Name()
		short := name.Short()

		var kind RefKind
		switch {
		case name.IsBranch():
			if short == "HEAD" {
				continue
			}
			kind = RefBranch
		case name.IsTag():
			kind = RefTag
		default:
			// skip notes/remote-tracking refs
			continue
		}

		parsed, ok := parseRefName(short)
		if !ok {
			continue
		}
		parsed.Kind = kind
		parsed.RefName = name
		refs = append(refs, parsed)
	}

	refCacheMu.Lock()
	refCache[repoURL] = refs
	refCacheMu.Unlock()

	return refs, nil
}

// parseRefName turns a short ref name into a RemoteRef with its versions
// filled in. ok is false for refs that do not carry a version at all
// ("main", "master", "develop", "HEAD", ...).
func parseRefName(short string) (RemoteRef, bool) {
	// tolerate the conventional "v" prefix on tags: v1.0.0 -> 1.0.0
	trimmed := short
	if strings.HasPrefix(trimmed, "v") || strings.HasPrefix(trimmed, "V") {
		trimmed = trimmed[1:]
	}

	versionPart, gamePart := splitRefName(trimmed)
	versionPart = strings.TrimSpace(versionPart)
	gamePart = strings.TrimSpace(gamePart)

	// Reserved branch names never carry a version.
	switch strings.ToLower(versionPart) {
	case "", "main", "master", "develop", "dev", "trunk", "head", "latest":
		return RemoteRef{}, false
	}

	v := version.Parse(versionPart)
	if len(v.Segments) == 0 {
		return RemoteRef{}, false
	}

	ref := RemoteRef{Name: short, Version: v}
	if gamePart != "" {
		gv := version.Parse(gamePart)
		if len(gv.Segments) > 0 {
			ref.GameVersion = gv
		}
	}
	return ref, true
}

// ResolveRef picks the git ref that satisfies spec.
//
// projectGameVersion is the game version of the local sculk-project; it is
// used as a soft preference when the caller did not pin one, so that
// `sculk add id-system@1.0.0` really does mean "1.0.0 for my GameVer".
//
// The returned bool is false when nothing pinned anything, which tells the
// caller to fall back to the repository's default branch.
func ResolveRef(repoURL string, spec LibrarySpec, projectGameVersion string) (RemoteRef, bool, error) {
	// Nothing pinned at all -> default branch.
	if !spec.WantsSpecificGameVersion() && spec.IsLatest() && spec.Version == "" {
		return RemoteRef{}, false, nil
	}
	if spec.IsLatest() && !spec.WantsSpecificGameVersion() && version.IsWildcard(spec.Version) {
		return RemoteRef{}, false, nil
	}

	refs, err := ListRemoteRefs(repoURL)
	if err != nil {
		return RemoteRef{}, false, err
	}
	if len(refs) == 0 {
		log.Printf("ℹ '%s' publishes no versioned branches or tags, using its default branch.", spec.Identifier)
		return RemoteRef{}, false, nil
	}

	verC := spec.VersionConstraint()
	gvC := spec.GameVersionConstraint()

	// 1. Exact ref-name match wins. Build the candidate spellings in order of
	//    preference and take the first that exists.
	if !spec.IsLatest() || spec.WantsSpecificGameVersion() {
		for _, candidate := range exactRefCandidates(spec, projectGameVersion) {
			if ref, ok := findRef(refs, candidate); ok {
				log.Printf("🎯 '%s' resolves to %s '%s'.", spec.Identifier, ref.Kind, ref.Name)
				return ref, true, nil
			}
		}
	}

	// 2. Fall back to constraint matching over every published ref.
	var matches []RemoteRef
	for _, ref := range refs {
		if !verC.Matches(ref.Version) {
			continue
		}
		if spec.WantsSpecificGameVersion() && !gvC.Matches(ref.GameVersion) {
			continue
		}
		matches = append(matches, ref)
	}

	if len(matches) == 0 {
		return RemoteRef{}, false, fmt.Errorf(
			"no published version of '%s' satisfies %s (available: %s)",
			spec.Identifier, describeSpec(spec, projectGameVersion), summariseRefs(refs),
		)
	}

	sortRefs(matches, projectGameVersion, spec)
	picked := matches[0]
	log.Printf("🎯 '%s' resolves to %s '%s' (best of %d matching refs).",
		spec.Identifier, picked.Kind, picked.Name, len(matches))
	return picked, true, nil
}

// exactRefCandidates lists the literal ref names to try first.
func exactRefCandidates(spec LibrarySpec, projectGameVersion string) []string {
	var out []string

	ver := strings.TrimSpace(spec.Version)
	gv := strings.TrimSpace(spec.GameVersion)

	if spec.IsLatest() {
		ver = ""
	}

	switch {
	case ver != "" && gv != "":
		out = append(out, refSpellings(ver, gv)...)
	case ver != "" && gv == "" && projectGameVersion != "":
		// "current GameVer" first, then the unqualified ref.
		out = append(out, refSpellings(ver, projectGameVersion)...)
		out = append(out, ver, "v"+ver)
	case ver != "":
		out = append(out, ver, "v"+ver)
	case ver == "" && gv != "":
		// "sculk add lib@/26.2" - mostly handled by the constraint pass, but
		// try the literal spelling too in case someone published exactly that.
		out = append(out, gv, "v"+gv)
	}

	return out
}

// refSpellings returns every accepted way of writing "version for gameVer" as
// a ref name, preferred spelling first.
func refSpellings(ver, gameVer string) []string {
	var out []string
	for _, sep := range refSeparators {
		out = append(out, ver+sep+gameVer, "v"+ver+sep+gameVer)
	}
	return out
}

// findRef looks up a literal ref name, branches first.
func findRef(refs []RemoteRef, name string) (RemoteRef, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return RemoteRef{}, false
	}
	// branch beats tag when both exist
	for _, r := range refs {
		if r.Kind == RefBranch && strings.EqualFold(r.Name, name) {
			return r, true
		}
	}
	for _, r := range refs {
		if r.Kind == RefTag && strings.EqualFold(r.Name, name) {
			return r, true
		}
	}
	return RemoteRef{}, false
}

// sortRefs orders candidates best-first.
//
// When the caller did not pin a game version, refs matching the local
// project's game version float to the top; within a group the highest version
// wins, then the highest game version.
func sortRefs(refs []RemoteRef, projectGameVersion string, spec LibrarySpec) {
	projectGV := version.Parse(projectGameVersion)
	prefersProjectGV := !spec.WantsSpecificGameVersion() && len(projectGV.Segments) > 0

	sort.SliceStable(refs, func(i, j int) bool {
		a, b := refs[i], refs[j]

		if prefersProjectGV {
			am := a.GameVersion.Equal(projectGV)
			bm := b.GameVersion.Equal(projectGV)
			if am != bm {
				return am
			}
		}

		if c := b.Version.Compare(a.Version); c != 0 {
			return c < 0
		}
		if c := b.GameVersion.Compare(a.GameVersion); c != 0 {
			return c < 0
		}
		// deterministic tie-break
		return a.Name < b.Name
	})
}

func describeSpec(spec LibrarySpec, projectGameVersion string) string {
	if s := spec.String(); s != spec.Identifier {
		return s
	}
	if projectGameVersion != "" {
		return "version * for GameVer " + projectGameVersion
	}
	return "*"
}

// summariseRefs renders the available refs for an error message.
func summariseRefs(refs []RemoteRef) string {
	seen := map[string]bool{}
	names := make([]string, 0, len(refs))
	for _, r := range refs {
		n := r.FullName()
		if seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
	}
	sort.Strings(names)
	const max = 12
	if len(names) > max {
		return strings.Join(names[:max], ", ") + fmt.Sprintf(", … (+%d more)", len(names)-max)
	}
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}
