declare const data: { stars: number };
export { data };

export default {
  async load() {
    const data = await (await fetch("https://api.github.com/repos/amir20/dozzle")).json();
    return { stars: data.stargazers_count };
  },
};
