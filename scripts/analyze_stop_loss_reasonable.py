#!/usr/bin/env python3
"""
分析0.3%止损距离是否合理
基于实际市场数据计算3分钟K线的波动性
"""

import json
import glob
import os
import re
from statistics import mean, stdev

def calculate_price_volatility(log_dir, num_logs=50):
    """计算实际3分钟K线的价格波动性"""
    
    log_files = sorted(glob.glob(os.path.join(log_dir, "decision_*.json")), reverse=True)[:num_logs]
    
    btc_volatilities = []
    eth_volatilities = []
    btc_atr_pct = []
    eth_atr_pct = []
    
    for log_file in log_files:
        try:
            with open(log_file, 'r', encoding='utf-8') as f:
                data = json.load(f)
            
            input_prompt = data.get('input_prompt', '')
            
            # 提取BTC数据
            btc_match = re.search(r'### 1\. BTCUSDT.*?Mid prices: \[([\d.,\s]+)\].*?ATR indicators.*?\[([\d.,\s]+)\]', input_prompt, re.DOTALL)
            if btc_match:
                prices_str = btc_match.group(1)
                atr_str = btc_match.group(2)
                
                prices = [float(p.strip()) for p in prices_str.split(',') if p.strip()]
                atr_values = [float(a.strip()) for a in atr_str.split(',') if a.strip()]
                
                if len(prices) >= 2:
                    # 计算价格范围
                    price_range = max(prices) - min(prices)
                    avg_price = mean(prices)
                    volatility_pct = (price_range / avg_price) * 100
                    btc_volatilities.append(volatility_pct)
                
                if atr_values:
                    # 计算ATR百分比
                    avg_price = mean(prices) if prices else 91000
                    atr_pct = (mean(atr_values) / avg_price) * 100
                    btc_atr_pct.append(atr_pct)
            
            # 提取ETH数据
            eth_match = re.search(r'### 2\. ETHUSDT.*?Mid prices: \[([\d.,\s]+)\].*?ATR indicators.*?\[([\d.,\s]+)\]', input_prompt, re.DOTALL)
            if eth_match:
                prices_str = eth_match.group(1)
                atr_str = eth_match.group(2)
                
                prices = [float(p.strip()) for p in prices_str.split(',') if p.strip()]
                atr_values = [float(a.strip()) for a in atr_str.split(',') if a.strip()]
                
                if len(prices) >= 2:
                    price_range = max(prices) - min(prices)
                    avg_price = mean(prices)
                    volatility_pct = (price_range / avg_price) * 100
                    eth_volatilities.append(volatility_pct)
                
                if atr_values:
                    avg_price = mean(prices) if prices else 3000
                    atr_pct = (mean(atr_values) / avg_price) * 100
                    eth_atr_pct.append(atr_pct)
        
        except Exception as e:
            continue
    
    return btc_volatilities, eth_volatilities, btc_atr_pct, eth_atr_pct

if __name__ == "__main__":
    log_dir = "/root/nofx/decision_logs_test/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728"
    
    btc_vol, eth_vol, btc_atr, eth_atr = calculate_price_volatility(log_dir, num_logs=50)
    
    print("=" * 80)
    print("3分钟K线波动性分析")
    print("=" * 80)
    
    if btc_vol:
        print(f"\nBTC 3分钟K线价格波动性（{len(btc_vol)}个样本）:")
        print(f"  平均波动范围: {mean(btc_vol):.3f}%")
        print(f"  最小波动: {min(btc_vol):.3f}%")
        print(f"  最大波动: {max(btc_vol):.3f}%")
        print(f"  标准差: {stdev(btc_vol):.3f}%")
        print(f"  中位数: {sorted(btc_vol)[len(btc_vol)//2]:.3f}%")
        print(f"  75%分位数: {sorted(btc_vol)[int(len(btc_vol)*0.75)]:.3f}%")
        print(f"  95%分位数: {sorted(btc_vol)[int(len(btc_vol)*0.95)]:.3f}%")
    
    if btc_atr:
        print(f"\nBTC ATR百分比（{len(btc_atr)}个样本）:")
        print(f"  平均ATR: {mean(btc_atr):.3f}%")
        print(f"  最小ATR: {min(btc_atr):.3f}%")
        print(f"  最大ATR: {max(btc_atr):.3f}%")
        print(f"  中位数: {sorted(btc_atr)[len(btc_atr)//2]:.3f}%")
    
    if eth_vol:
        print(f"\nETH 3分钟K线价格波动性（{len(eth_vol)}个样本）:")
        print(f"  平均波动范围: {mean(eth_vol):.3f}%")
        print(f"  最小波动: {min(eth_vol):.3f}%")
        print(f"  最大波动: {max(eth_vol):.3f}%")
        print(f"  标准差: {stdev(eth_vol):.3f}%")
        print(f"  中位数: {sorted(eth_vol)[len(eth_vol)//2]:.3f}%")
        print(f"  75%分位数: {sorted(eth_vol)[int(len(eth_vol)*0.75)]:.3f}%")
        print(f"  95%分位数: {sorted(eth_vol)[int(len(eth_vol)*0.95)]:.3f}%")
    
    if eth_atr:
        print(f"\nETH ATR百分比（{len(eth_atr)}个样本）:")
        print(f"  平均ATR: {mean(eth_atr):.3f}%")
        print(f"  最小ATR: {min(eth_atr):.3f}%")
        print(f"  最大ATR: {max(eth_atr):.3f}%")
        print(f"  中位数: {sorted(eth_atr)[len(eth_atr)//2]:.3f}%")
    
    print("\n" + "=" * 80)
    print("0.3%止损距离合理性分析")
    print("=" * 80)
    
    if btc_atr:
        avg_atr = mean(btc_atr)
        recommended_min = avg_atr * 2.5  # 2.5倍ATR作为安全边际
        print(f"\nBTC分析:")
        print(f"  平均ATR: {avg_atr:.3f}%")
        print(f"  建议最小止损（2.5×ATR）: {recommended_min:.3f}%")
        print(f"  当前设置: 0.3%")
        if 0.3 >= recommended_min:
            print(f"  ✅ 0.3% ≥ {recommended_min:.3f}%，设置合理")
        else:
            print(f"  ⚠️  0.3% < {recommended_min:.3f}%，可能偏窄")
        print(f"  建议范围: {max(0.3, recommended_min):.3f}% - 0.5%")
    
    if eth_atr:
        avg_atr = mean(eth_atr)
        recommended_min = avg_atr * 2.5
        print(f"\nETH分析:")
        print(f"  平均ATR: {avg_atr:.3f}%")
        print(f"  建议最小止损（2.5×ATR）: {recommended_min:.3f}%")
        print(f"  当前设置: 0.3%")
        if 0.3 >= recommended_min:
            print(f"  ✅ 0.3% ≥ {recommended_min:.3f}%，设置合理")
        else:
            print(f"  ⚠️  0.3% < {recommended_min:.3f}%，可能偏窄")
        print(f"  建议范围: {max(0.3, recommended_min):.3f}% - 0.5%")
    
    print("\n" + "=" * 80)
    print("结论与建议")
    print("=" * 80)
    print("""
考虑因素：
1. 3分钟K线波动性：需要覆盖正常市场噪音
2. 执行延迟：3-30秒的执行延迟可能导致价格变化
3. 滑点：订单执行时的价格滑点
4. 安全边际：建议使用2-3倍ATR作为最小止损

建议：
- 如果平均ATR < 0.12%，0.3%是合理的（2.5×0.12% = 0.3%）
- 如果平均ATR > 0.12%，建议提高到0.35-0.4%
- 在震荡市场或高波动率环境中，应使用更宽的止损（0.4-0.5%）
""")

