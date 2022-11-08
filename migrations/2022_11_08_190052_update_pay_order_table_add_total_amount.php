<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class UpdatePayOrderTableAddTotalAmount extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('pay_order', function (Blueprint $table) {
            $table->string('amount')->comment('预收金额')->change();
            $table->string('amount_total')->comment('总金额')->nullable();
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
