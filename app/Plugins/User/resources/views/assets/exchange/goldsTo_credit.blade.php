{{--<b>{{get_options('wealth_money_name', '余额') . ' >>> ' . get_options('wealth_golds_name', '金币')}}</b>--}}
@php($proportion = get_options('wealth_how_many_golds_to_credit',10))
兑换比例: 1 {{get_options('wealth_golds_name', '金币')}} >>>  {{$proportion}} {{get_options('wealth_credit_name', '积分')}}
<div class="mb-2">
    @php($user = \App\Plugins\User\src\Models\User::query()->with('Options')->find(auth()->id()))
    @if(count(explode('.',$user->Options->golds))>1)
        @php( $dc = \Hyperf\Utils\Str::after((string)$user->Options->golds,'.'))
        @php($dc = (int)\Hyperf\Utils\Str::length($dc))
    @else
        @php($dc = 0)
    @endif
    当前 {{get_options('wealth_golds_name', '金币')}}数量: <b class="text-red">{{$user->Options->golds}}</b>，
    最多能兑换 <b class="text-red">{{intval($user->Options->golds*$proportion)}}</b> {{get_options('wealth_credit_name', '积分')}}
    @if(round($user->Options->golds - (intval($user->Options->golds*$proportion))/$proportion,$dc)!=0)
        有 {{round($user->Options->golds - (intval($user->Options->golds*$proportion))/$proportion,$dc)}} {{get_options('wealth_golds_name', '金币')}} 不能兑换
    @endif
</div>


<div class="mb-2">
    <label for="" class="form-label"></label>
    <input type="number" min="0" step="1" max="{{intval($user->Options->golds*$proportion)}}" class="form-control" v-model="data.goldsTo_credit_num" placeholder="输入兑换的{{get_options('wealth_credit_name', '积分')}}数量" required>
</div>