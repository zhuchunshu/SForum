@extends("App::app")
@section('title','【'.$order->id.'】订单信息')
@section('content')
    <div class="row row-cards">
        <div class="col-md-12">
            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">订单信息</h3>
                    <div class="card-actions">
                        <a href="/users/{{auth()->id()}}.html?m=users_home_menu_7">返回订单列表</a>
                    </div>
                </div>
                <div class="card-body">
                    @if($order->status==="待支付" || $order->status ==="待付款" || $order->status==="未支付" || $order->status==="未付款" )
                        <div class="datagrid">
                            <div class="datagrid-item">
                                <div class="datagrid-title">订单号</div>
                                <div class="datagrid-content">{{$order->id}}</div>
                            </div>
                            <div class="datagrid-item">
                                <div class="datagrid-title">订单标题</div>
                                <div class="datagrid-content">{{$order->title}}</div>
                            </div>
                            <div class="datagrid-item">
                                <div class="datagrid-title">预收金额</div>
                                <div class="datagrid-content">{{$order->amount}} CNY</div>
                            </div>
                            <div class="datagrid-item">
                                <div class="datagrid-title">交易状态</div>
                                <div class="datagrid-content">
                      <span class="status status-{{pay()->generate_html()->status_color_name($order->status)}}">
                        {{$order->status}}
                      </span>
                                </div>
                            </div>
                        </div>
                        <div id="user-order-show-paying">
                            <form action="" @@submit.prevent="submit">
                                <div class="row justify-content-center mt-4">
                                    <div v-if="!paid" class="col-12 text-center">
                                        <h3 class="card-title" v-if="qrcode_url">请扫码支付</h3>
                                        <h3 class="card-title" v-else>选择支付方式</h3>
                                        <div v-if="qrcode_url">
                                            <img :src="qrcode_url" alt="">
                                        </div>
                                        <span class="text-muted" v-if="pay_url">
                                            或点此链接支付:<a :href="pay_url">@{{ pay_url }}</a>
                                        </span>
                                        <div class="form-selectgroup" v-if="!qrcode_url">
                                            @foreach(pay()->get_enabled_data() as $id=>$data)
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
                                            @endforeach
                                        </div>
                                        <div v-if="!qrcode_url" class="mt-3">
                                            <button class="btn btn-primary">立即支付</button>
                                        </div>
                                    </div>
                                    <div v-else class="col-12 text-center">
                                        支付成功,请刷新当前页面
                                    </div>
                                </div>
                            </form>
                        </div>
                    @else
                        <div class="datagrid">
                            @if($order->id)
                                <div class="datagrid-item">
                                    <div class="datagrid-title">订单号</div>
                                    <div class="datagrid-content">{{$order->id}}</div>
                                </div>
                            @endif
                            @if($order->trade_no)
                                <div class="datagrid-item">
                                    <div class="datagrid-title">交易单号</div>
                                    <div class="datagrid-content">{{$order->trade_no}}</div>
                                </div>
                            @endif
                            @if($order->amount)
                                <div class="datagrid-item">
                                    <div class="datagrid-title">预收金额</div>
                                    <div class="datagrid-content">{{$order->amount}} {{get_options('wealth_money_unit_name','元')}}</div>
                                </div>
                            @endif
                            @if($order->payer_total)
                                <div class="datagrid-item">
                                    <div class="datagrid-title">用户实付金额</div>
                                    <div class="datagrid-content">{{$order->payer_total}} {{get_options('wealth_money_unit_name','元')}}</div>
                                </div>
                            @endif

                            @if($order->payer_total)
                                <div class="datagrid-item">
                                    <div class="datagrid-title">实收金额(订单总金额)</div>
                                    <div class="datagrid-content">{{$order->amount_total}} {{get_options('wealth_money_unit_name','元')}}</div>
                                </div>
                            @endif
                            @if($order->title)
                                <div class="datagrid-item">
                                    <div class="datagrid-title">订单标题</div>
                                    <div class="datagrid-content">{{$order->title}}</div>
                                </div>
                            @endif
                            <div class="datagrid-item">
                                <div class="datagrid-title">交易状态</div>
                                <div class="datagrid-content">
                      <span class="status status-{{pay()->generate_html()->status_color_name($order->status)}}">
                        {{$order->status}}
                      </span>
                                </div>
                            </div>
                            <div class="datagrid-item">
                                <div class="datagrid-title">创建时间</div>
                                <div class="datagrid-content">{{$order->created_at}}</div>
                            </div>
                            <div class="datagrid-item">
                                <div class="datagrid-title">最后更新时间</div>
                                <div class="datagrid-content">{{$order->updated_at}}</div>
                            </div>
                            @if(!@is_string(pay()->get_pay_plugin_data(json_decode($order->payment_method,true)[0],json_decode($order->payment_method,true)[1])))
                                <div class="datagrid-item">
                                    <div class="datagrid-title">支付方式</div>
                                    <div class="datagrid-content">
                                        <!-- Download SVG icon from http://tabler-icons.io/i/check -->
                                        @php($payment=pay()->get_pay_plugin_data(json_decode($order->payment_method,true)[0],json_decode($order->payment_method,true)[1]))
                                        <img data-bs-toggle="tooltip" data-bs-html="true" data-bs-placement="top"
                                             title="<b>{{$payment['name']}}</b>:{{$payment['description']}}"
                                             style="height:30px;max-width:125px" src="{{$payment['logo']}}"
                                             alt="{{$payment['name']}}">
                                    </div>
                                </div>
                            @endif
                        </div>
                    @endif
                </div>
            </div>
        </div>
    </div>

@endsection

@section('header')
    <div class="page-wrapper">
        <div class="container-xl">
            <!-- Page title -->
            <div class="page-header d-print-none">
                <div class="row align-items-center">
                    <div class="col">
                        <!-- Page pre-title -->
                        <div class="page-pretitle">
                            Overview
                        </div>
                        <h2 class="page-title">
                            订单信息
                        </h2>
                    </div>
                </div>
            </div>
        </div>
    </div>
@endsection

@section('scripts')
    <script>
        var order_id = "{{$order->id}}";
    </script>
    <script src="{{mix('plugins/User/js/order.js')}}"></script>
@endsection