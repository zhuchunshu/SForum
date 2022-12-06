<div class="row row-cards">
    <div class="col-md-12">
        <div class="card border-0">
            <div class="card-header">
                <h3 class="card-title">{{$user->username}} 的订单</h3>
                <div class="card-actions">
                    <form action="{{ request()->getRequestUri()."?".core_http_build_query(request()->all(),[])}}"
                          method="get">
                        @foreach(request()->all() as $k=>$v)
                            @if($k!=='trade_no')
                                <input type="hidden" name="{{$k}}" value="{{$v}}"/>
                            @endif
                        @endforeach
                        <input type="hidden" name="page" value="1"/>
                        <div class="row g-2">
                            <div class="col">
                                <input type="text" value="{{request()->input('trade_no')}}" name="trade_no"
                                       class="form-control" placeholder="输入订单号或交易号">
                            </div>
                            <div class="col-auto">
                                <button type="submit" class="btn btn-icon" aria-label="Button">
                                    <!-- Download SVG icon from http://tabler-icons.io/i/search -->
                                    <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24"
                                         viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                         stroke-linecap="round" stroke-linejoin="round">
                                        <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                                        <circle cx="10" cy="10" r="7"/>
                                        <line x1="21" y1="21" x2="15" y2="15"/>
                                    </svg>
                                </button>
                            </div>
                        </div>
                    </form>
                </div>
            </div>
            <div class="card-body">
                @if(\App\Plugins\Core\src\Models\PayOrder::query()->where('user_id',$user->id)->exists())
                    @if(request()->input('trade_no'))
                        @php($payOrderPage = \App\Plugins\Core\src\Models\PayOrder::query()->where([['user_id',$user->id],['id','like','%'.request()->input('trade_no')."%"]])->orWhere([['user_id',$user->id],['trade_no','like','%'.request()->input('trade_no')."%"]])->orderByDesc('created_at')->paginate(15))
                    @else
                        @php($payOrderPage = \App\Plugins\Core\src\Models\PayOrder::query()->where('user_id',$user->id)->orderByDesc('created_at')->paginate(15))
                    @endif
                    <div class="table-responsive">
                        <table class="table table-vcenter table-nowrap">
                            <thead>
                            <tr>
                                <th>订单号</th>
                                <th>订单标题</th>
                                <th>预收金额</th>
                                <th>订单状态</th>
                                <th>交易单号</th>
                                <th>交易金额</th>
                                <th>支付方式</th>
                                <th>创建时间</th>
                                <th></th>
                            </tr>
                            </thead>
                            <tbody>
                            @if($payOrderPage->count())
                                @foreach($payOrderPage as $order)
                                    <tr>
                                        <td><a href="/user/order/{{$order->id}}.order">{{$order->id}}</a></td>
                                        <td class="text-muted">
                                            {{$order->title}}
                                        </td>
                                        <td>{{$order->amount}} {{get_options('wealth_money_unit_name','元')}}</td>
                                        <td>
                                            <span class="badge badge-outline {{pay()->generate_html()->status_class_text($order->status)}}">{{$order->status}}</span>
                                        </td>
                                        <td class="text-muted">@if($order->trade_no) {{\Hyperf\Utils\Str::limit($order->trade_no,8)}} @else
                                                <span class="badge badge-outline {{pay()->generate_html()->status_class_text($order->status)}}">{{$order->status}}</span> @endif
                                        </td>
                                        <td>@if($order->amount_total){{$order->amount_total}} {{get_options('wealth_money_unit_name','元')}} @else <span
                                                    class="badge badge-outline {{pay()->generate_html()->status_class_text($order->status)}}">{{$order->status}}</span> @endif
                                        </td>
                                        @if($order->payment_method) @php($payment = pay()->get_pay_plugin_data((string)json_decode($order->payment_method,true)[0],json_decode($order->payment_method,true)[1])) @else @php($payment = '获取失败') @endif
                                        <td class="text-center">@if(!is_string($payment)) <img
                                                    style="max-width:70px;max-height:30px" src="{{$payment['logo']}}"
                                                    alt="{{$payment['name']}}">  @else 获取失败 @endif</td>
                                        <td>{{$order->created_at}}</td>
                                    </tr>
                                @endforeach
                            @else
                                <tr>
                                    <td>暂无数据</td>
                                    <td>暂无数据</td>
                                    <td>暂无数据</td>
                                    <td>暂无数据</td>
                                    <td>暂无数据</td>
                                    <td>暂无数据</td>
                                    <td>暂无数据</td>
                                    <td>暂无数据</td>
                                    <td>暂无数据</td>
                                </tr>
                            @endif
                            </tbody>
                        </table>
                    </div>
                    @if($payOrderPage->count())
                        <div class="mt-3 mb-0">
                            {!! make_page($payOrderPage) !!}
                        </div>
                    @endif
                @else
                    <div class="empty">
                        <div class="empty-header">404</div>
                        <p class="empty-title">暂无更多结果</p>
                    </div>
                @endif
            </div>
        </div>
    </div>
</div>
