@extends('app')

@section('title','【'.$order->id.'】的订单信息')

@section('content')
    <div class="row row-cards">
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">订单信息</h3>
            </div>
            <div class="card-body">
                <div class="datagrid">
                    <div class="datagrid-item">
                        <div class="datagrid-title">订单号</div>
                        <div class="datagrid-content">{{$order->id}}</div>
                    </div>
                    <div class="datagrid-item">
                        <div class="datagrid-title">交易单号</div>
                        <div class="datagrid-content">{{$order->trade_no}}</div>
                    </div>
                    <div class="datagrid-item">
                        <div class="datagrid-title">预收金额</div>
                        <div class="datagrid-content">{{$order->amount}} CNY</div>
                    </div>
                    <div class="datagrid-item">
                        <div class="datagrid-title">用户实付金额</div>
                        <div class="datagrid-content">{{$result['payer_total']}} CNY</div>
                    </div>
                    <div class="datagrid-item">
                        <div class="datagrid-title">实收金额(订单总金额)</div>
                        <div class="datagrid-content">{{$result['amount_total']}} CNY</div>
                    </div>
                    <div class="datagrid-item">
                        <div class="datagrid-title">订单创建者</div>
                        <div class="datagrid-content">
                            <div class="d-flex align-items-center">
                                <a href="/users/{{$order->user->id}}.html"><span
                                            class="avatar avatar-xs me-2 avatar-rounded"
                                            style="background-image: url({{super_avatar($order->user)}})"></span></a>
                                {{$order->user->username}}
                            </div>
                        </div>
                    </div>
                    <div class="datagrid-item">
                        <div class="datagrid-title">订单标题</div>
                        <div class="datagrid-content">{{$order->title}}</div>
                    </div>
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
                    @if($result['success_time'])
                        <div class="datagrid-item">
                            <div class="datagrid-title">付款时间</div>
                            <div class="datagrid-content">
                                <!-- Download SVG icon from http://tabler-icons.io/i/check -->
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon text-green" width="24" height="24"
                                     viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                     stroke-linecap="round" stroke-linejoin="round">
                                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                                    <path d="M5 12l5 5l10 -10"/>
                                </svg>
                                {{$result['success_time']}}
                            </div>
                        </div>
                    @endif
                    @if(!@is_string(pay()->get_pay_plugin_data(json_decode($order->payment_method,true)[0],json_decode($order->payment_method,true)[1])))
                        <div class="datagrid-item">
                            <div class="datagrid-title">支付方式</div>
                            <div class="datagrid-content">
                                <!-- Download SVG icon from http://tabler-icons.io/i/check -->
                                @php($payment=pay()->get_pay_plugin_data(json_decode($order->payment_method,true)[0],json_decode($order->payment_method,true)[1]))
                                <img data-bs-toggle="tooltip" data-bs-html="true" data-bs-placement="top" title="<b>{{$payment['name']}}</b>:{{$payment['description']}}" style="height:30px;max-width:125px" src="{{$payment['logo']}}" alt="{{$payment['name']}}">
                            </div>
                        </div>
                    @endif
                </div>
            </div>
        </div>
    </div>
@endsection

@section('headerBtn')
    <a href="/admin/Pay" class="btn btn-primary">订单列表</a>
@endsection