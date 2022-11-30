<div class="row row-cards">
    <div class="col-md-12">
        <div class="card border-0">
            <div class="card-header">
                <h3 class="card-title">账户充值</h3>
            </div>
            <div class="card-body" id="user-data-money-recharge">
                <div v-if="paid" class="text-center">
                    <h3 class="card-title">支付成功!</h3>
                </div>
                <div v-if="!paid">
                    <div v-if="!qrcode_url">
                        <form method="post" @@submit.prevent="submit">
                            <div class="mb-3">
                                <label for="" class="form-label">
                                    充值金额
                                </label>
                                <input type="number" v-model="amount" min="0.00" max="1000.00" step="0.01"
                                       class="form-control" required>
                            </div>
                            <div class="mb-3">
                                <label for="" class="form-label">验证码</label>
                                <div class="input-group">
                                    <input type="text" v-model="captcha" class="form-control" placeholder="captcha"
                                           autocomplete="off" required>
                                    <span class="input-group-link">
                        <img class="captcha" src="{{captcha()->inline()}}" alt=""
                             onclick="this.src='/captcha?id='+Math.random()">
                    </span>
                                </div>
                            </div>
                            <div class="mb-3">
                                <label for="" class="form-label">选择支付方式</label>
                                <div class="form-selectgroup">
                                    @foreach(pay()->get_enabled_data() as $id=>$data)
                                        @if((int)$id!==0)
                                            <label class="form-selectgroup-item">
                                                <input type="radio" v-model="payment"
                                                       value='[{{$id}},"{{$data['ename']}}"]'
                                                       class="form-selectgroup-input">
                                                <span class="form-selectgroup-label"><!-- Download SVG icon from http://tabler-icons.io/i/home -->
                                                      <span class="avatar avatar-sm">
                                                          <img src="{{$data['icon']}}"
                                                               alt="{{$data['name']}},{{$data['description']}}">
                                                      </span>
                                                      {{$data['name']}}
                                                    </span>
                                            </label>
                                        @endif
                                    @endforeach
                                </div>
                            </div>
                            <div class="mb-1">
                                <button :disabled="btn_disabled" class="btn btn-primary">立即支付</button>
                            </div>
                        </form>
                    </div>
                </div>
                <div v-if="qrcode_url" class="text-center">
                    <h3 class="card-title" v-if="qrcode_url">请扫码支付</h3>
                    <img :src="qrcode_url" alt="" style="max-width:250px">
                    <div class="text-muted" v-if="pay_url">
                                            或点此链接支付:<a :href="pay_url">@{{ pay_url }}</a>
                                        </div>
                </div>
            </div>
        </div>
    </div>
</div>

@section('scripts')
    <script>var user_id = {{$user->id}}</script>
    <script src="{{mix("plugins/Core/js/user.js")}}"></script>
    <script src="{{mix('plugins/User/js/order.js')}}"></script>
@endsection