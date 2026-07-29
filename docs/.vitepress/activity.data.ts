declare const data: {
  stars: number;
  pulls: number;
  contributors: number;
  release: { version: string; date: string } | null;
};
export { data };

// The contributors endpoint has no total count, so ask for one per page and read
// the last page number out of the Link header.
async function fetchContributorCount() {
  const res = await fetch("https://api.github.com/repos/amir20/dozzle/contributors?per_page=1&anon=false");
  const match = res.headers.get("link")?.match(/[?&]page=(\d+)>; rel="last"/);
  return match ? parseInt(match[1], 10) : 0;
}

async function fetchLatestRelease() {
  const res = await fetch("https://api.github.com/repos/amir20/dozzle/releases/latest");
  const json = await res.json();
  if (!json.tag_name) return null;
  return { version: json.tag_name, date: json.published_at };
}

export default {
  async load() {
    const urls = [
      "https://api.github.com/repos/amir20/dozzle",
      "https://hub.docker.com/v2/namespaces/amir20/repositories/dozzle",
    ];

    const [repo, hub, contributors, release] = await Promise.all([
      ...urls.map((url) => fetch(url).then((res) => res.json())),
      // Never fail the build over a rate-limited or flaky API call.
      fetchContributorCount().catch(() => 0),
      fetchLatestRelease().catch(() => null),
    ]);

    return {
      stars: repo.stargazers_count,
      pulls: hub.pull_count,
      contributors,
      release,
    };
  },
};
