@extends('Topic::ShortCode.buy.layout')
@section('buy_content')
    你需要支付 <span class="text-primary">{{ $amount }}</span> {{get_options('wealth_credit_name','积分')}} 才能查看此内容
@endsection