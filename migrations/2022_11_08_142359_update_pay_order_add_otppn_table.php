<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class UpdatePayOrderAddOtppnTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('pay_order', function (Blueprint $table) {
            $table->string('trade_no')->comment('交易单号')->nullable();
            $table->string('payer_total')->comment('实收金额')->nullable();
            $table->string('payment_method')->comment('支付方式')->nullable();
            $table->longText('notify_result')->comment('回调数据')->nullable();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('pay_order', function (Blueprint $table) {
            //
        });
    }
}
