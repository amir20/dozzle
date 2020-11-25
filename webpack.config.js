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
  devtool: argv.mode !== "production" ? "inline-cheap-source-map" : false,
  entry: ["./assets/main.js", "./assets/styles.scss"],
  output: {
    path: path.resolve(__dirname, "./static"),
    filename: "[name].js",
    publicPath: argv.mode === "production" ? "{{ .Base }}" : "/",
  },
  module: {
    rules: [
      {
        test: /\.vue$/,
        loader: "vue-loader",
      },
      {
        test: /\.m?js$/,
        exclude: /(node_modules|bower_components)/,
        use: {
          loader: "babel-loader",
          options: {
            presets: ["@babel/preset-env"],
          },
        },
      },
      {
        test: /\.(sass|scss|css)$/,
        use: [
          {
            loader: MiniCssExtractPlugin.loader,
          },
          "css-loader",
          {
            loader: "postcss-loader",
            options: {
              postcssOptions: {
                plugins: [require("autoprefixer")],
              },
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
  devServer: {
    port: 8081,
    inline: true,
    hot: true,
    open: true,
    historyApiFallback: true,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
      },
    },
  },
});
