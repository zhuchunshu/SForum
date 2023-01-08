@extends('app')

@section('title','订单管理')

@section('content')
    <div class="row row-cards">
        <div class="card">
            <div class="card-header">
                <div class="card-actions">
                    <form action="/admin/Pay" method="get">
                        <label class="form-label">订单搜索</label>
                        <div class="row g-2">
                            <div class="col">
                                <input type="text" value="{{request()->input('trade_no')}}" name="trade_no" class="form-control" placeholder="输入订单号或交易号">
                            </div>
                            <div class="col-auto">
                                <button type="submit" class="btn btn-icon" aria-label="Button">
                                    <!-- Download SVG icon from http://tabler-icons.io/i/search -->
                                    <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><circle cx="10" cy="10" r="7" /><line x1="21" y1="21" x2="15" y2="15" /></svg>
                                </button>
                            </div>
                        </div>
                    </form>
                </div>
            </div>
            <div class="card-body">
                <div id="table-default" class="table-responsive">
                    <table class="table table-vcenter table-nowrap">
                        <thead>
                        <tr>
                            <th><a href="?{{ core_http_build_query(request()->all(),['where_column'=>'id','_orderBy' => core_default($_orderBy,'ASC')]) }}" class="table-sort">订单号</a></th>
                            <th>订单标题</th>
                            <th><a href="?{{ core_http_build_query(request()->all(),['where_column'=>'user_id','_orderBy' => core_default($_orderBy,'ASC')]) }}" class="table-sort">创建者</a></th>
                            <th>预收金额</th>
                            <th><a href="?{{ core_http_build_query(request()->all(),['where_column'=>'trade_no','_orderBy' => core_default($_orderBy,'ASC')]) }}" class="table-sort">交易单号</a></th>
                            <th>实收金额</th>
                            <th>总交易金额</th>
                            <th>支付方式</th>
                            <th><a href="?{{ core_http_build_query(request()->all(),['where_column'=>'status','_orderBy' => core_default($_orderBy,'ASC')]) }}" class="table-sort">状态</a></th>
                            <th><a href="?{{ core_http_build_query(request()->all(),['where_column'=>'created_at','_orderBy' => core_default($_orderBy,'ASC')]) }}" class="table-sort">创建时间</a></th>
                            <th><a href="?{{ core_http_build_query(request()->all(),['where_column'=>'updated_at','_orderBy' => core_default($_orderBy,'ASC')]) }}" class="table-sort">更新时间</a></th>
                            <th></th>
                        </tr>
                        </thead>
                        <tbody class="table-tbody">
                        @if(!$page->count())
                            <tr>
                                <td class="sort-id">无结果</td>
                                <td class="sort-store">无结果</td>
                                <td class="sort-name">无结果</td>
                                <td class="sort-log">无结果</td>
                                <td class="sort-data">无结果</td>
                                <td class="sort-data">无结果</td>
                                <td class="sort-data">无结果</td>
                                <td class="sort-data">无结果</td>
                                <td class="sort-data">无结果</td>
                                <td class="sort-data">无结果</td>
                                <td class="sort-data">无结果</td>
                            </tr>
                        @else
                            @foreach($page->items() as $data)
                                <tr>
                                    <td class="sort-id">@if($data->trade_no)<a href="/admin/Pay/{{$data->trade_no}}/order">{{$data->id}}</a> @else {{$data->id}} @endif</td>
                                    <td class="sort-title">{{$data->title}}</td>
                                    <td class="sort-user"><a href="/users/{{$data->user->id}}.html"><span class="avatar avatar-sm" style="background-image: url({{super_avatar($data->user)}})"></span></a></td>
                                    <td class="sort-amount">{{$data->amount}}</td>
                                    <td class="sort-trade_no"><a href="/admin/Pay/{{$data->trade_no}}/order">{{$data->trade_no}}</a></td>
                                    <td class="sort-payer_total">{{$data->payer_total}}</td>
                                    <td class="sort-amount_total">{{$data->amount_total}}</td>
                                    <td class="sort-payment_method">@if($data->payment_method) <span class="badge badge-outline text-cyan">{{pay()->get_pay_plugin_data((string)json_decode($data->payment_method,true)[0],json_decode($data->payment_method,true)[1])['name']}}</span> @else 获取失败 @endif</td>
                                    <td class="sort-status"><span class="badge badge-outline {{pay()->generate_html()->status_class_text($data->status)}}">{{$data->status}}</span></td>
                                    <td class="sort-date" data-date="{{strtotime($data['created_at'])}}">{{$data['created_at']}}</td>
                                    <td class="sort-date" data-date="{{strtotime($data['updated_at'])}}">{{$data['updated_at']}}</td>
{{--                                    <td><a href="/admin/server/logger/{{$data['_id']}}.html">查看</a></td>--}}
                                </tr>
                            @endforeach
                        @endif
                        </tbody>
                    </table>
                </div>
            </div>
            @if($page->count())
                {!! make_page($page) !!}
            @endif
        </div>
    </div>
@endsection

