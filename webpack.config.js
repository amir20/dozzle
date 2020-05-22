const path = require("path");
const { VueLoaderPlugin } = require("vue-loader");
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const HtmlWebpackPlugin = require("html-webpack-plugin");
const WebpackPwaManifest = require("webpack-pwa-manifest");

module.exports = (env, argv) => ({
  stats: { children: false, entrypoints: false, modules: false },
  performance: {
    maxAssetSize: 350000,
    maxEntrypointSize: 600000,
  },
  devtool: argv.mode === "development" ? "inline-cheap-source-map" : false,
  entry: ["./assets/main.js", "./assets/styles.scss"],
  output: {
    path: path.resolve(__dirname, "./static"),
    filename: "[name].js",
    publicPath: "{{ .Base }}",
  },
  module: {
    rules: [
      {
        test: /\.vue$/,
        loader: "vue-loader",
      },
      {
        test: /\.(sass|scss|css)$/,
        use: [
          MiniCssExtractPlugin.loader,
          {
            loader: "css-loader",
            query: {
              importLoaders: 1,
            },
          },
          {
            loader: "postcss-loader",
            options: {
              ident: "postcss",
              plugins: (loader) => [
                require("postcss-import")(),
                require("postcss-cssnext")({
                  features: {
                    customProperties: { warnings: false },
                  },
                }),
              ],
            },
          },
          "sass-loader",
        ],
      },
    ],
  },
  plugins: [
    new VueLoaderPlugin(),
    new MiniCssExtractPlugin(),
    new HtmlWebpackPlugin({
      hash: true,
      template: "assets/index.ejs",
      scriptLoading: "defer",
      favicon: "assets/favicon.svg",
    }),
    new WebpackPwaManifest({
      name: "Dozzle Log Viewer",
      short_name: "Dozzle",
      theme_color: "#222",
      background_color: "#222",
      display: "standalone",
    }),
  ],
  resolve: {
    alias: {
      vue$: "vue/dist/vue.runtime.esm.js",
    },
    extensions: ["*", ".js", ".vue", ".json"],
  },
});
