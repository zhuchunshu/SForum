<div class="card card-body">
  <div class="row">
      <div class="mb-3 col-lg-4">
          <label class="form-label">CloudFlare Turnstile 站点密钥</label>
          <input type="text" class="form-control" v-model="data.admin_captcha_cloudflare_turnstile_website_key">
      </div>
      <div class="mb-3 col-lg-4">
          <label class="form-label">CloudFlare Turnstile 密钥</label>
          <input type="text" class="form-control" v-model="data.admin_captcha_cloudflare_turnstile_key">
          <small>这个是服务端用的</small>
      </div>
      <div class="mb-3 col-lg-4 align-center">
          <label for="" class="form-label">CloudFlare Turnstile链接</label>
          <p>
              <a href="https://dash.cloudflare.com/?to=/:account/turnstile">https://dash.cloudflare.com/?to=/:account/turnstile</a>
          </p>
      </div>
  </div>
</div>